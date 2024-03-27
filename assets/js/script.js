const handlerChangeShownColumn = (el) => {
    let key = el.id.split("-")[1]

    let minColumn = document.querySelector("#mac_min-"+key)
    let maxColumn = document.querySelector("#mac_max-"+key)

    if (minColumn.style.display === "none") {
        minColumn.style.display = "table-cell"
        maxColumn.style.display = "none"
    } else {
        maxColumn.style.display = "table-cell"
        minColumn.style.display = "none"
    }
}

function handlerTransformColumn(el) {
    let col = el.parentNode
    let value = col.innerText
    let btn = document.createElement("button")
    let key = col.id.split("-")[1]

    col.innerHTML = `<input type='text' value="${value}" id="input-${key}"/> `

    btn.setAttribute("onclick", `handlerSendChange(${key})`)
    btn.innerHTML = "Сохранить"
    col.append(btn)
}

function handlerSendChange(key) {
    let value = document.querySelector(`#input-${key}`).value
    let ip
    if (window.location.href.includes("dgs")) {
        ip = window.location.href.split("snmp/dgs/")[1]
    } else if (window.location.href.includes("eltex")) {
        ip = window.location.href.split("snmp/eltex/")[1]
    }

    let options = {
        method: "POST",
        body: JSON.stringify({
            Index: Number(key),
            Description: value
        })
    }

    fetch("/snmp/dgs/change_port_description/"+ip, options)
        .then(response => response.json())
        .then(data => {
            if (data) {
                let col = document.querySelector(`#description-${key}`)
                col.innerHTML = `${value} <img onclick="handlerTransformColumn(this)" src="/snmp/assets/public/pen.svg" alt="O">`
            } else {
                alert("не удалось изменить описание")
            }
        })
        .catch(error => console.error(error))
}