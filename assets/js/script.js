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