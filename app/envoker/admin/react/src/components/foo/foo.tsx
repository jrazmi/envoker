import React from "react";

export const useFoo = () => {
  const [foo, setFoo] = React.useState("bar");

  return { foo, setFoo };
};

export default useFoo;
