---
layout: ../layouts/Post.astro
title: Analysing AWS VPC Flow logs with Python and Pandas
date: 2023-02-09
tags: aws vpc python pandas data analysis
author: Simon Bracegirdle
description: Where did the datas come from, where did they go? AWS VPC flow log analysis with Python, Pandas and Jupyter notebooks to understand AWS network costs.
image: panda
---

## Context

Recently, I encountered an AWS EC2 bill that was higher than expected and I suspected that traffic flowing in and out of the [NAT Gateway](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-nat-gateway.html) was the culprit. In this post, I will share my journey of using Python and its powerful data analytics ecosystem to analyze VPC flow logs and gain insights into AWS networking costs.

Before diving into a solution, I always strive to have a good understanding of the problem to avoid wasting precious engineering time optimizing the wrong thing. In this case, I needed to gather more information about the traffic flow within the private network. To achieve this, I leveraged [VPC flow logs](https://docs.aws.amazon.com/vpc/latest/userguide/flow-logs.html), which contain a record of network activity within an AWS VPC.


## Analysis

### Python notebook in VS Code

I've had some experience doing simple data analysis in Python before, specifically with [Pandas](https://pandas.pydata.org/), [Matplotlib](https://matplotlib.org/), [Numpy](https://numpy.org/), and other popular data science libraries, so it made sense that I leverage those skills rather than trying to learn something like AWS Athena.

I went for a Jupyter notebook, which is a popular development environment for data analysis. It allows you to run Python code in small chunks known as cells, which can be interwoven with text, visualisations and other content. With the [VS Code Python](https://marketplace.visualstudio.com/items?itemName=ms-python.python) extension, you can treat any Python file as a pseudo-notebook by marking the start of a cell with the `#%%` character string. You can then execute that cell directly inside VS Code and get instant feedback.

For example, here's a cell to import the libraries we need:

```py
#%%
import pandas as pd
import os
import boto3
```

With that up and running, I moved on to retrieving the data set.

### Pulling down logs from AWS S3.

In this case, the VPC flow logs were already stored in AWS S3, so I was able to download them in compressed format directly.

I wanted a subset of the logs for one day, I didn't need an entire day or month, as long as the sample was representative of the whole. This will help to keep the transfer and computation time down too.

The cell below downloads the first 20 files from a S3 path and stores them locally:

```py
#%%
s3 = boto3.resource('s3')

# Download all the files from a S3 path
def download_files_from_s3(bucket, s3_path, local_path):
    if not os.path.exists(os.path.dirname(local_path)):
        os.makedirs(os.path.dirname(local_path))

    bucket = s3.Bucket(bucket)
    count = 0
    for obj in bucket.objects.filter(Prefix=s3_path):
        if count > 20: # First 20 files please
            break
        count += 1

        # Strip any path separators from the file name
        filename = obj.key.split('/')[-1]

        # Download the file if it doesn't already exist locally
        if not os.path.exists(local_path + filename):
            print('Downloading', obj.key)
            bucket.download_file(obj.key, local_path + filename)
        else:
            print('Skipping', obj.key)

download_files_from_s3('my-vpc-logs-bucket', 'path-to-the-logs/2023/02/07', 'data/')
```

### Loading the data into a dataframe

Now that I had my data stored locally, I wanted to get it into a data structure in memory that would make analysis of the data easier. The most common data structure for analysis like this in Python is the [Pandas data frame](https://pandas.pydata.org/docs/reference/api/pandas.DataFrame.html), which is a two dimensional structure that allows for easy aggregation, grouping, filtering and visualisation. Data frames are even more powerful when used with other libraries in the Python ecosystem such as matplotlib and numpy.

The following cell reads the first 20 files it finds in a directory, un-compresses them and appends them to the primary data frame:

```py
# %%
# Process log.gz files into a single dataframe
def process_log_files(local_path):
    df = pd.DataFrame()
    # Only process first X files
    count = 0
    for file in os.listdir(local_path):
        if file.endswith(".gz") and count < 20:
            print('Processing', file)
            df = df.append(pd.read_csv(local_path + file, compression='gzip', header=None, sep=' ', names=['version', 'account_id', 'interface_id', 'src_addr', 'dst_addr', 'src_port', 'dst_port', 'protocol', 'packets', 'bytes', 'start', 'end', 'action', 'log_status']))
            count += 1
    return df

df = process_log_files('data/')

# %%
# Print the first rows of the table to verify the data looks right
df.head()
```

Now that there's a sampling of the data in memory, we can commence analysis.

### Looking for the largest destination of data

The first question I had for the data was to find out which host was receiving the most bytes.

That meant converting the bytes column into a numeric format that we can use in aggregations:

```py
df['bytes'] = pd.to_numeric(df['bytes'], errors='coerce')
```

Then I grouped by destination address, tallied the bytes, and sorted in descending order:

```py
# %%
# Group by destination address, sum bytes as a new column
result = df.groupby(['dst_addr']).sum()['bytes'].reset_index()
# Sort by bytes descending
result = result.sort_values(by=['bytes'], ascending=False)
result.head()
```

There was one IP address that stood out by a large margin, so I was curious to learn more about it. It fell within the VPC CIDR, so I queried ENI's in AWS to see if there was a match:

```py
# %%
# Query the resource attached to a destination address
def query_resource(dst_addr):
    client = boto3.client('ec2')
    response = client.describe_network_interfaces(
        Filters=[
            {
                'Name': 'addresses.private-ip-address',
                'Values': [
                    dst_addr,
                ]
            },
        ],
    )
    return response

print(query_resource('IP_ADDRESS_HERE'))
```

This returned a large amount of metadata about the ENI, but the description made it clear that the interface belonged to the NAT Gateway, confirming the hunch I mentioned earlier.

### Looking for the largest sender of data

I wanted to understand where this NAT traffic was originating from, as I hoped it would lead to optimisations that can trim down the AWS bill.

I grouped by source address, where the destination address was the NAT gateway, then tallied the bytes and sorted in descending order:

```py
# %%
# Find the source that sends the most bytes to the destination in question
result = df[df['dst_addr'] == 'NAT_IP_ADDRESS_HERE'].groupby(['src_addr']).sum()['bytes'].reset_index().sort_values(by=['bytes'], ascending=False)
result.head()
```

This revealed IP addresses that weren't in the VPC — the traffic was coming from the interwebs.

I installed the [`ipwhois`](https://pypi.org/project/ipwhois/) library, which would allow me to lookup metadata about an IP address, such as which ISP or network it belongs to:

```py
#%%
# pip3 install ipwhois
from ipwhois import IPWhois, IPDefinedError

#%%
# Use ipwhois to lookup metadata about the IP address
def call_ipwhois(ip):
    # Catch IPDefinedError
    try:
        result = IPWhois(ip).lookup_rdap(depth=1)
    except IPDefinedError as e:
        result = None

    return result

print(call_ipwhois('SUSPECT_IP_HERE'))
```

I did this for the top 25 source addresses to see if there any patterns in ISP or network. This is slow since each API call takes a second or two. If I was going to do this more than once I'd optimise it, but this is a once-off task so I didn't bother.


```py
#%%
# Create in memory cache for ipwhois results to make re-querying faster
ipwhoiscache = {}

#%%
count = 0

# Iterate over dataframe
for index, row in result.iterrows():
    # Call ipwhois for first 25 rows
    if count < 25:
        # Print index
        print(count, index, row['src_addr'])

        if row['src_addr'] in ipwhoiscache:
            ipmeta = ipwhoiscache[row['src_addr']]
        else:
            ipmeta = call_ipwhois(row['src_addr'])
            ipwhoiscache[row['src_addr']] = ipmeta

        if ipmeta is not None:
            result.loc[index, 'network'] = ipmeta['network']['name']

        count += 1

# Print result, show top 25
result.head(25)
```

I grouped by network name, tallied the bytes and sorted descending to find the network sending the most bytes to the NAT:

```py
# %%
# Group by network name and sum bytes
result_network = result.groupby(['network']).sum()['bytes'].reset_index().sort_values(by=['bytes'], ascending=False)
result_network.head(25)
```

It turns out that most of the traffic was coming from the network `AT-88-Z`, owned by Amazon Technologies Inc. In other words, this is traffic flowing between AWS services and the NAT Gateway.

## Retrospective

This simple analysis provided enough information to identify which AWS resource was sending this data, which led me to make config changes that drastically reduced the AWS networking bill.

I think this demonstrates the power of Pandas and Python for quick analysis jobs like this. If I was going to productionise this analysis, for example with a regular report to management, or if I needed to crunch larger amounts of data, I'd consider using something like AWS SageMaker or AWS Athena. But for this particular ad-hoc case with a smaller data set, Pandas and Python in a locally running notebook was the perfect choice.

Thanks for reading, please get in contact with me on [Twitter](https://twitter.com/bracegirdle_me) or [LinkedIn](https://www.linkedin.com/in/simon-bracegirdle/) if you have any comments or questions.
