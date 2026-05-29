document.addEventListener("DOMContentLoaded", () => {

    const minusButton = document.querySelector(".minus");
    const plusButton = document.querySelector(".plus");

    BACKEND.onEvent("timer", ({current_time})=>{
        document.querySelector("#timer").textContent = current_time
    })

    document.querySelector("#timer").addEventListener("click", ()=>{
        BACKEND.call("eval", {}, null)

    })

    minusButton.addEventListener("click", () => {

        const value = getCurrentValue();
        BACKEND.call("sub", { value }, (({data, err}) => {
            if (err) {
                console.error(err)
                return
            }
            setCurrentValue(data.value)
        }));
        
    })

    plusButton.addEventListener("click", () => {
        const value = getCurrentValue();
        BACKEND.call("add", { value }, (({data, err}) => {
            if (err) {
                console.error(err)
                return
            }
            setCurrentValue(data.value)
        }));

    })
})

function getCurrentValue() {
    const digits = [...document.querySelectorAll(".digit")];
    const value = parseInt(digits.map(digit => digit.textContent).join(""));
    console.log(({value}))
    return value
}

/**
 * @param {number} newValue
 */
function setCurrentValue(newValue) {
  if (newValue < 0) {
    document.querySelector(".sign").textContent = "-"
  } else {
    document.querySelector(".sign").textContent = ""
  }

  newValue = Math.abs(newValue);

  let valuesChar = String(newValue).split(""); // FIX

  const digits = [...document.querySelectorAll(".num")];

  while (valuesChar.length < digits.length) {
    valuesChar.unshift("0");
  }

  console.log(({ valuesChar }));

  valuesChar.forEach((char, index) => { // FIX
    digits[index].textContent = char;
  });
}