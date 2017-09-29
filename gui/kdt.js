const execa = require('execa');
const path = require('path');

const startUploadRegex = /start upload/gm
const percentRegex = /percent=(\d+\.\d+)%/gm;
const finishedRegex = /file is uploaded/gm;

//error information
const brokenRegex = /query file info no info/gm;
const duplicatedRegex = /file is already uploaded/gm;
const failedOpenRegex = /failed to open input file/gm;

var kdtBinary = "kdt";
if (process.platform === "windows") {
	kdtBinary = "kdt.exe";
}
//kdt client --remoteaddr 127.0.0.1:4000  --datashard 0 --parityshard 0  --sndwnd 8192 
function kdt(remoteaddr, datashard, parityshard, key, crypt, filename, progressCallback) {
	var uploader = execa(path.join(__dirname, "vendor", kdtBinary),
			['client',
			'--key', key,
			'--remoteaddr', remoteaddr, 
			'--datashard', datashard,
			'--parityshard', parityshard,
			'--sndwnd', '8192',
			'--crypt', crypt,
			filename,
			]);

	var wrappedPromise = new Promise((resolve, reject) => {
		let success = false;
		let errorType = "";
		if (key.trim() == "") {
			key = "none";
		}
		if (crypt == "") {
			crypt = "none";
		}
		console.log(path.join(__dirname, "vendor", kdtBinary));


		//kdt use stderr 
		uploader.stderr.on("data", data=>{
			data = data.toString('utf8').trim();
			let startUploadMatch = startUploadRegex.exec(data);
			if (startUploadMatch) {
			}
			let percentMatch = percentRegex.exec(data);
			if (percentMatch) {
				progressCallback(percentMatch[1]);
			}

			let finishedMatch = finishedRegex.exec(data);
			if (finishedMatch) {
				success = true;
			}
			let brokenMatch = brokenRegex.exec(data);
			if (brokenMatch) {
				success = false;
				errorType = "network broken";
			}

			let duplicatedMatch = duplicatedRegex.exec(data);
			if (duplicatedMatch) {
				success = false;
				errorType = "duplicated file";
			}

			let failedOpenMatch = failedOpenRegex.exec(data);
			if (failedOpenMatch) {
				success = false;
				errorType = "failed to open file: " + filename;
			}
		});

		uploader.on('close', (code,signal) => {
			if (success === true) {
				resolve();
			} else {
				if (signal == 'SIGTERM')
					reject("killed");
				else
					reject(errorType);
			}
		});
	});

	wrappedPromise.kill = function(){
		uploader.kill();
	}
	return wrappedPromise;
	
}
exports.kdt = kdt;
