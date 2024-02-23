---
layout: ../layouts/Post.astro
title: React anti-patterns that lead to unnecessary complexity
date: 2024-02-23
tags: software react complexity testing front-end
author: Simon Bracegirdle
description: Where we look at common React anti-patterns such as unnecessary use of useEffect, unnecessary state, premature memoisation, and large inline functions.
image: headache
---

As called out by the legend of the craft, [Grug](https://grugbrain.dev), complexity can be the bane of your existence as a software developer. Unnecessary complexity leads to code that is hard to understand and reason about, and makes it easy to introduce bugs.

I have been doing React long enough to know that it's not exempt from having complex, hard to read code. Whether it's old school Redux, class components, or newer hooks and server components, complexity can creep it at any point if we're not careful.

What patterns can we look out for that might flag that a problem is ahead? In this post i'll cover what I think are some common anti-patterns and indicators that your React code might be more complex than it needs to be.

## Anti-pattern 1 — Unnecessary effects

The react paradigm is all about writing reactive code — code that produces output (rendered elements) in response to input (props, state). `useEffect` allows us to do some side action that doesn't directly impact the rendered output. This could be updating the window title when a certain prop changes, or focusing an input field on first render.

It's an escape hatch, so we need to be cautious with how we use it to avoid issues. Let's look at an example of how that can happen:

```jsx
function BadUseEffectComponent() {
  const {loading, error, data} = useQuery(GET_DATA);
  const [records, setRecords] = useState([]);

  useEffect(() => {
    if (data) {
      setRecords(data.records);
    }
  }, [data]);

  const handleEdit = (id, newValue) => {
    setRecords(records.map(record => 
      record.id === id ? { ...record, value: newValue } : record
    ));
  };

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return data.records.map(({ id, value }) => (
    <div key={id}>
      <input 
        type="text" 
        value={value} 
        onChange={e => handleEdit(id, e.target.value)} 
      />
    </div>
  ));
}
```

Here we have a component that queries data from a GraphQL API (`useQuery`), and then uses an effect to copy that data into state (`records`). When the user edits a record (`input` `onChange`), we override the state value for that data record (`handleEdit`).

I can understand why people want to do this; they want a single variable containing the values they're going to render, it's a model that makes sense.

But, the presence of the `useEffect` here can add make it harder to read because we have to understand the conditions the effect fires and the flow-on effect it has on state and rendering. Oversights in following this logic can lead to bugs, of which I have experienced too many.

Returning to the code above, if the query was to run again, such as due to props changing, the effect could fire and override the user's edited data! The use of an effect to copy data into state has created a bug.

Here's how we could re-write the code without an effect:

```jsx
function BetterComponent() {
  const {loading, error, data} = useQuery(GET_DATA);
  const [editedRecords, setEditedRecords] = useState({});

  const handleEdit = (id, newValue) => {
    setEditedRecords({ ...editedRecords, [id]: newValue });
  };

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return data.records.map(({ id, value }) => (
    <div key={id}>
      <input 
        type="text" 
        value={editedRecords[id] || value} 
        onChange={e => handleEdit(id, e.target.value)} 
      />
    </div>
  ));
}
```

This time we still have state to hold the user's edited values, but we do not copy the back-end data into state. We can combine the back-end data with the user state in the render body. This is also easier to test because our function no longer has less effects. If the back-end query is re-run, we retain our unsaved edited values, removing a critical bug.

My recommendation here is to avoid `useEffect` as much as possible. In general, don't use it to set derived state, and don't use it do mapping. Instead of using it to fetch back-end data, look at a robust query library that provides hooks like React Query, SWR, or Apollo client.

Sometimes `useEffect` is necessary, but consider it a last resort when all other options aren't possible.


### Extreme variant — effect chain hell

To take the above to the extreme, chained effects with interdependencies can combine to create the ultimate in complexity hell:

```jsx
function ThisIsHell({ propA, propB }) {
  const { data, loading, error } = useQuery(SOME_QUERY);
  const [state, setState] = useState(null);
  const [derivedState, setDerivedState] = useState(null);
  const [finalState, setFinalState] = useState(null);

  // First useEffect based on Apollo query result
  useEffect(() => {
    if (!loading && data) {
      setState(data.someField);
    }
  }, [data, loading]);

  // Second useEffect based on the state set by the first useEffect
  useEffect(() => {
    if (state) {
      setDerivedState(`Derived: ${state}`);
    }
  }, [state]);

  // Third useEffect based on the state set by the second useEffect and propA
  useEffect(() => {
    if (derivedState && propA) {
      setFinalState(`${derivedState} and ${propA}`);
    }
  }, [derivedState, propA]);

  // Fourth useEffect based on the state set by the third useEffect and propB
  useEffect(() => {
    if (finalState && propB) {
      console.log(`Final state: ${finalState} and ${propB}`);
    }
  }, [finalState, propB]);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return <div>{finalState}</div>;
}
```

The above is a contrived example, but respresents a real world problem. Each of the effects are partially dependent on each other to create spaghetti code that is difficult to follow. Code like this is going to be impossible to understand, hard to test, and riddled with bugs.

I think this can be the result of overcomplicating the problem space in our head, which is easy to do when we're solving a non-trivial problem. A useful idea here might be to take a step away from the code, return to it fresh and look for alternative designs that lead to simpler code.


## Anti-pattern 2 — Unnecessary state

State is an important concept in React, allowing us to track values entered by the user before we're ready to send them to the back-end for persistence. But, a common issue is that it gets overused. Let's look at an example of that:

```jsx
function UnnecessaryState() {
  const [value1, setValue1] = useState('');
  const [value2, setValue2] = useState('');
  const [sum, setSum] = useState(0);

  const handleValue1Change = (e) => {
    setValue1(e.target.value);
    setSum(parseInt(e.target.value) + parseInt(value2));
  };

  const handleValue2Change = (e) => {
    setValue2(e.target.value);
    setSum(parseInt(value1) + parseInt(e.target.value));
  };

  return (
    <div>
      <input type="number" value={value1} onChange={handleValue1Change} />
      <input type="number" value={value2} onChange={handleValue2Change} />
      <p>The sum is: {sum}</p>
    </div>
  );
}
```

Here we have two state values, which change when the user updates the two number inputs. We also have a sum state, which update when either of the two values change. Then we show the sum below the two inputs.

But, we don't need to put `sum` in state at all, since we can calculate it in on the fly in our render:

```jsx
function SumInBody() {
  const [value1, setValue1] = useState('');
  const [value2, setValue2] = useState('');

  const handleValue1Change = (e) => {
    setValue1(e.target.value);
  };

  const handleValue2Change = (e) => {
    setValue2(e.target.value);
  };

  return (
    <div>
      <input type="number" value={value1} onChange={handleValue1Change} />
      <input type="number" value={value2} onChange={handleValue2Change} />
      <p>The sum is: {value1 + value2}</p>
    </div>
  );
}
```

Again this is a contrived example, but as components get complex it's easy for this pattern to creep into code and cause issues. For example, what if we add a third number value, and forget to update the `sum` state in that change handler. Putting data in state unnecessarily opens up our code for bugs when it gets modified later on.

In general we don't need to put derived data in state, we should prefer to use simple inline statements, or move the mapping logic into a separate function that we call from our component:

```jsx
function sum(value1, value2) {
  return value1 + value2;
}

/// ...

<p>The sum is: {sum(value1, value2)}</p>
```

`sum` is now easier to test since it's a pure function that returns a value based on some input, without any side effects.

In the worst case scenario, we can memoise `sum`, but as we'll discuss in the next section, we should be hesitant to do that.

## Side note — prefer state in URL

When you do need to use state for holding values the user has entered, it's often a good idea to hold that value in the URL query parameters, instead of plain `useState`. The reason for this is the user can then share the link with colleagues, friends, or your technical support in case they encounter an issue.

An example of this could be to hold the `searchTerm` in the URL, after the user has typed in a search bar.

One useful library in React is `use-query-params`, which provides some useful hooks:

```jsx
function MySearchComponent() {
  const [searchTerm, setSearchTerm] = useQueryParam('searchTerm', StringParam);

  const handleChange = (event) => {
    setSearchTerm(event.target.value);
  };

  return (
    <div>
      <input type="text" value={searchTerm || ''} onChange={handleChange} />
    </div>
  );
```

## Anti-pattern 3 — Premature memoisation

Memoisation is a powerful tool that builds on the idea of caching to prevent re-runs of a function unless a given set of dependencies change. If they don't change then react returns a cached value instead, potentially saving computation.

But, I think we are overrusing this tool in the React community — we should start by putting the computation in the component body — we can add memoisation later when it's needed.

Let's look at an example of premature memoisation:

```jsx
function PrematureMemo() {
  const {loading, error, data} = useQuery(GET_DATA);

  const myData = useMemo(() => 
    data?.data?.map(item => ({
    ...item,
    value: item.value * 2,
    }))
  , [data]);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return (
    <div>
      {myData?.map(item => (
        <p key={item.id}>Data: {item.value}</p>
      ))}
    </div>
  );
}
```

In this component, we have a `useQuery` (from Apollo GraphQL client) hook that we use to query data from our back-end. We then have a `useMemo` for performing some mapping operation on the resulting data and memoising it. We then render our elements based on that mapped data.

Instead we could have re-written the above like so:

```jsx
function SimpleMapping() {
  const { loading, error, data } = useQuery(GET_DATA);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return (
    <div>
      {data?.data?.map(item => (
        <p key={item.id}>Data: {item.value * 2}</p>
      ))}
    </div>
  );
}
```

The biggest change here is that we have removed the memo, and do the mapping in the render body instead.

Some might ask; "But that's not efficient, it'll be re-calculated on each render!". But, an O(n) mapping operation isn't necessarily computationally significant. In this context, we're talking about a handful of entries, which even slower devices can compute fast.

The other assumption is that rendering is happening all the time, which it generally isn't unless either state changes, props change, or the parent is re-rendered. The rendering lifecycle of React already acts a kind of memoisation, and we should leverage that before adding another layer.

By adding memoisation prematurely we could be adding a lot of unnecessary noise to our code, or even bugs if we don't get our dependency array right (like if we forgot `[data]` in the first example). Instead, observe real world performance, and only when performance is unsatisfactory should we look at optimisation.

That isn't to say we should write inefficient code by default — but don't assume any kind of loop or mapping is going to be slow.


## Anti-pattern 4 — Lots of large inline functions

This one is more of a readability problem as components get larger. Having a lot of inline functions can make a mess of your function, making it hard to follow the logic and trace the key data flow. Let's look at the following:

```jsx
const LargeInlineFunction = () => {
  const [response, setResponse] = React.useState(null);

  return (
    <div>
      <button onClick={() => {
        fetch('https://api.example.com/data')
          .then(response => response.json())
          .then(data => {
            // Perform some complex transformations on the data
            let transformedData = data;
            for (let i = 0; i < data.length; i++) {
              transformedData[i] = {
                ...data[i],
                extraProperty: 'extraValue'
              };
            }

            setResponse(transformedData);
          })
          .catch(error => console.error(error));
      }}>Do something</button>
      {response ? response.map(item => <div key={item.id}>{item.name}</div>)}
    </div>
  );
};
```

This particular example isn't too bad because it's a small component, but if you can imagine a component hundreds of lines long, with half a dozen large inline functions, it'll be hard to read and follow, more so when you add state and effects into the mix.

It's also hard to write tests for those functions liek this since they're buried inside the component, we'd need to mock out a bunch of things to get the code to trigger.

As a habit, I find moving these out into separate functions a good idea:

```jsx
// Separate function for fetching and transforming data
async function fetchData() {
  const response = await fetch('https://api.example.com/data');
  const data = await response.json();
}

function transformData(data) {
  // Perform some complex transformations on the data
  let transformedData = [];
  for (let i = 0; i < data.length; i++) {
    transformedData[i] = {
      ...data[i],
      extraProperty: 'extraValue'
    };
  }

  return transformedData;
}

const FunctionsMovedOut = () => {
  const [data, setData] = React.useState(null);

  return (
    <div>
      <button onClick={() => {
        fetchData()
          .then(transformData)
          .then(setData)
          .catch(console.error);
      }}>Do something</button>
      {data ? data.map(item => <div key={item.id}>{item.name}</div>) : 'Loading...'}
    </div>
  );
};
```

This means we can now write tests for `transformData`, without the hassle of mocking, since it's a pure function that produces output from input, without any side effects.


## Conclusion

Simplicity is a virtue in software development, and unnecessary complexity is going to be a impediment. In this post we've had a look at anti-patterns that can be a cause of this in my experience.

We explored the pitfalls of overusing `useEffect`, we discussed the unnecessary state usage, and encouraged developers to calculate derived data in the render body or use separate functions for mapping logic.

By avoiding these anti-patterns, I hope it puts you on the path of simpler, more readable, and more maintable React code.

I'd be keen to hear from you if you have any thoughts on patterns that help or cause harm in your experience.

Cheers.