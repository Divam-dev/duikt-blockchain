let web3
let contract
let depositSubscription = null
let transferSubscription = null
const processedEvents = new Set()

const contractAddress = '0x3E9e7e74945b844093335338eaf07F5Ed3737D5c';
const contractABIPath = "/static/abi.json";

async function connectWallet() {
    if (window.ethereum) {
        try{
            const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
            web3 = new Web3(window.ethereum);
            document.getElementById("status").innerText = `Connected: ${accounts[0]}`;

            const response = await fetch(contractABIPath);
            const json = await response.json();
            const abi = json.abi || json;
            console.log(abi);
            contract = new web3.eth.Contract(abi, contractAddress);

            subscribeToEvents();
        } catch (error) {
            console.error("connectWallet error: ", error);
            alert("Користувачеві відмовлено в доступі або сталася помилка");
        }
    } else {
        alert("Будь ласка, встановіть MetaMask!")
    }
}

async function deposit() {
    if (!contract) return alert("Contract not initialized. Please connect MetaMask first!");
    try {
        const accounts = await web3.eth.getAccounts();
        const amount = document.getElementById("depositAmount").value;

        if (!amount || isNaN(amount)) {
            alert("Please enter a valid amount!");
            return;
        }

        await contract.methods.deposit().send({
            from: accounts[0],
            value: web3.utils.toWei(amount, "ether"),
            gas: 300000
        });
        alert("Deposit successful!");
        document.getElementById("depositAmount").value = "";
    } catch (error) {
        console.error("Deposit error: ", error);
        alert("Deposit failed!" + (error.message || "Unknown error"));
    }
}

function displayBalance(message) {
    const balanceElement = document.getElementById("balanceResult");
    balanceElement.style.display = "block";
    balanceElement.innerText = message;
}

async function getBalance() {
    try {
        if (!contract) {
            displayBalance("Помилка: Спочатку підключіть гаманець!");
            return;
        }

        const address = document.getElementById("userAddress").value.trim();

        if (!web3.utils.isAddress(address)) {
            displayBalance("Помилка: Введіть коректну адресу Ethereum");
            return
        }

        const balance = await contract.methods.getBalance(address).call();
        const formattedBalance = web3.utils.fromWei(balance, "ether");
        displayBalance(`Баланс: ${formattedBalance} ETH`);

    } catch (error) {
        console.error("Error getting balance: ", error);
        displayBalance(`Помилка: ${error.message || "Не вдалося отримати баланс"}`)
    }
}

async function transfer(){
    if (!contract) return alert("Contract not initialized. Please connect MetaMask first!");
    try{
        const accounts = await web3.eth.getAccounts();
        const to = document.getElementById("transferTo").value.trim();
        const amount = document.getElementById("transferAmount").value;

        if(!web3.utils.isAddress(to)) {
            alert("Please enter a valid address!");
            return;
        }
        if (!amount || isNaN(amount)) {
            alert("Please enter a valid amount!");
            return;
        }

        const amountWei = web3.utils.toWei(amount, "ether");
        await contract.methods.transfer(to, amountWei).send({
            from: accounts[0]
        });
        alert("Transfer successful!");
        document.getElementById("transferTo").value = "";
        document.getElementById("transferAmount").value = "";
    } catch (error) {
        console.error("Transfer error: ", error);
        alert("Transfer failed!" + (error.message || "Unknown error"));
    }
}

function logEvent(message) {
    const eventLog = document.getElementById("eventLog");

    const emptyMessage = eventLog.querySelector(".text-muted");
    if (emptyMessage) {
        emptyMessage.remove();
    }

    const eventItem = document.createElement("div");
    eventItem.className = "event-item";
    eventItem.innerHTML = `<small>${new Date().toLocaleString()}: </small>${message}`;
    eventLog.prepend(eventItem);
}

function unsubscribeFromEvents() {
    if (depositSubscription) {
        depositSubscription.unsubscribe();
        depositSubscription = null;
    }
    if (transferSubscription) {
        transferSubscription.unsubscribe();
        transferSubscription = null;
    }
}

function subscribeToEvents() {
    if (!contract) {
        console.warn("Contract не ініціалізовано");
        return;
    }

    unsubscribeFromEvents();
    processedEvents.clear();

    logEvent("Підписка на eventi контракту активована");

    depositSubscription = contract.events.Deposit({
        fromBlock: 'latest'
    })
        .on("data", event => {
            const eventKey = `${event.transactionHash}-${event.logIndex}`;

            if (processedEvents.has(eventKey)) {
                return;
            }
            processedEvents.add(eventKey);

            console.log("Deposit event:", event);
            const amount = web3.utils.fromWei(event.returnValues.amount, "ether");
            logEvent(`✅ Депозит: ${event.returnValues.sender} поповнив на ${amount} ETH`);
        })
        .on("error", error => {
            console.error("Помилка підписки на депозити: ", error);
        });

    transferSubscription = contract.events.Transfer({
        fromBlock: 'latest'
    })
        .on("data", event => {
            const eventKey = `${event.transactionHash}-${event.logIndex}`;

            if (processedEvents.has(eventKey)) {
                return;
            }
            processedEvents.add(eventKey);

            console.log("Transfer event:", event);
            const amount = web3.utils.fromWei(event.returnValues.amount, "ether");
            logEvent(`✅ Переказ: ${event.returnValues.from} → ${event.returnValues.to} (${amount} ETH)`);
        })
        .on("error", error => {
            console.error("Помилка підписки на трансфери: ", error);
        });
}