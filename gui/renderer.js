const electron = require('electron');
const { ipcRenderer } = electron;

var nodeConsole = require('console');
var myConsole = new nodeConsole.Console(process.stdout, process.stderr);


var inputFile = document.querySelector('#uploader input');
var inputFileMessage= document.querySelector("#uploader p");
var inputFileForm = document.querySelector("#uploader form");
var optionSubmitButton = document.getElementById('submitConfig');
var optionDiv = document.getElementById('options');
var uploadBtn = document.getElementById('uploadBtn');
var optionBtn = document.querySelector("#optionBtn button");


[ 'drag', 'dragstart', 'dragend', 'dragover', 'dragenter', 'dragleave', 'drop' ].forEach( function( event )
{
    document.addEventListener( event, function( e )
    {
        // preventing the unwanted behaviours
        e.preventDefault();
        e.stopPropagation();
    });
});

['dragover', 'dragenter'].forEach(function(e){
    inputFileForm.addEventListener(e, ()=>{
        inputFileForm.classList.add("is_dragover");
    })
});

['dragleave', 'dragend', 'drop'].forEach(function(e){
    inputFileForm.addEventListener(e, ()=>{
        inputFileForm.classList.remove("is_dragover");
    })
})

function baseName(p) {

    var seperator = '/'
    if (p[0] === "\\") {
        seperator = "\\";
    }
    return new String(p).substring(p.lastIndexOf(seperator) + 1);
}

var fileNamePath = "";
//drop file or select the file;
inputFileForm.addEventListener( 'drop', function( e )
{
    const { path } = e.dataTransfer.files[0];
    inputFileMessage.innerHTML = baseName(path);
    fileNamePath = path;

    resetAlertProgressBarAndButton();
});

inputFile.addEventListener("change", (e)=>{

    const { path } = inputFile.files[0];
    inputFileMessage.innerHTML = baseName(path);
    fileNamePath = path;

    resetAlertProgressBarAndButton();

});


//TODO animation
//if saved, hide the option div
optionSubmitButton.addEventListener("click",function(e){
    e.preventDefault();


    //get Configure from HTML;
    var opts = {};
    for (var el in optionGroup) {
        opts[el] = optionGroup[el].value;
    }

    //save the config/
    saveConfig(opts);
    //hide the options
    optionDiv.style.display = "none";
    optionBtn.style.display = "inline-block"

});

//click the option button to show the options
//and hide itself;
optionBtn.addEventListener("click", function(){
    //hide the option button itself
    renderOptionHtml();

    optionDiv.style.display = "block";
    optionBtn.style.display = "none";
})

var isUploading  = false;

//upload the file through kdt
uploadBtn.addEventListener("click", function(){
    //read config from local storage
    var config  = loadConfig();
    //START UPLOADING
    if(fileNamePath !== "" && isUploading == false) {
        ipcRenderer.send("file:submit", {'path':fileNamePath, 'config':config});
        //turn upload button to stop button
        showInfoAlert("Uploading file " + baseName(fileNamePath));
        isUploading = true;
        uploadBtn.innerHTML = "<i class='fa fa-pause' </i> Pause";
    //STOP UPLOADING
    } else if (isUploading == true) {
        //TODO;
    } else if (!fileNamePath) {
        showErrorAlert("not specify a filename");
    }
});

var progressDiv = document.querySelector("div .progress-bar");
var messagezoneDiv = document.querySelector("#messagezone");

ipcRenderer.on("file:progress", (event, progress)=>{
    var p = Math.floor(progress);
    progressDiv.style.width = `${p}%`;
});

ipcRenderer.on("file:result", (event, msg)=>{
    myConsole.log("result:" + msg);
    if (msg == "success") {
        progressDiv.style.width = "100%";
        //messagezoneDiv.innerHTML = "success";
        showSuccessAlert("Upload Success");
    } else {
        showErrorAlert(msg);
    }

    uploadBtn.innerHTML = "<i class='fa fa-cloud-upload'></i> Upload One More?";
    fileNamePath = "";
    inputFileMessage.innerHTML = "<p>drag your files here </p>";
    isUploading = false;

});


function loadConfig() {
    var keys = ['endpoint', 'key', 'crypt', 'datashard', 'parityshard'];
    var opts = {};
    for (let i = 0 ; i < keys.length ; i++) {
        // myConsole.log(keys[i]);
        if (keys[i] === 'datashard' || keys[i] === 'parityshard') {
            opts[keys[i]] = window.localStorage.getItem(keys[i]) || 0;
        } else {
            opts[keys[i]] = window.localStorage.getItem(keys[i]) || '';
        }
        //myConsole.log(opts);
    }
    return opts;
}

function saveConfig(opts) {
    for (let k in opts) {
        window.localStorage.setItem(k, opts[k]);
    }
}

/*
saveConfig({
    'endpoint':"127.0.0.1:4000",
    'key':'',
    'crypt':'',
    'datashard':0,
    'parityshard':0
});
*/

//myConsole.log(loadConfig());


//set html options
var endpoint = document.querySelector('div input[name=endpoint]');
var key = document.querySelector('div input[name=key]');
var crypt = document.querySelector('div select[name=crypt]');
var datashard = document.querySelector('div input[name=datashard]');
var parityshard = document.querySelector('div input[name=parityshard]');

var optionGroup = {
    "endpoint":endpoint, 
    "key":key, 
    "crypt":crypt, 
    "datashard":datashard,
    "parityshard":parityshard
}


function renderOptionHtml() {
    var opts = loadConfig();
    endpoint.value = opts['endpoint'];
    key.value = opts['key'];
    crypt.value = opts['crypt'];
    datashard.value = opts['datashard'];
    parityshard.value = opts['parityshard'];
}
//type
function showSuccessAlert(msg) {
    messagezoneDiv.classList = [];
    messagezoneDiv.classList.add
    messagezoneDiv.classList.add("alert-success");
    messagezoneDiv.innerHTML = msg;
}

function showErrorAlert(msg) {
    messagezoneDiv.classList = [];
    messagezoneDiv.classList.add("alert-danger");
    messagezoneDiv.innerHTML = msg;
}

function showInfoAlert(msg) {
    messagezoneDiv.classList = [];
    messagezoneDiv.classList.add("alert-info");
    messagezoneDiv.innerHTML = msg;
}

function resetAlertProgressBarAndButton(){
    messagezoneDiv.classList = [];
    messagezoneDiv.innerHTML = "";
    progressDiv.style.width = "0%";
    uploadBtn.innerHTML = "<i class='fa fa-cloud-upload'></i> Upload";
}
