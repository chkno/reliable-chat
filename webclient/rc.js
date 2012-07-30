var servers = ['chkno.net', 'localhost'];

var session = Math.random();
var seen = {};

function rcserverbase(server) {
	// Add the default port if server doesn't contain a port number already
	if (server.indexOf(":") == -1) {
		return "http://" + server + ":21059";
	} else {
		return "http://" + server;
	}
}

function rcchangeserverstatus(server, new_status) {
	var statusbar = document.getElementById("status");
	var spans = statusbar.getElementsByTagName("span");
	for (i in spans) {
		if (spans[i].firstChild && 'data' in spans[i].firstChild && spans[i].firstChild.data == server) {
			spans[i].setAttribute("class", new_status);
		}
	}
}

function rcaddmessagetohistory(message) {
	var d = document.createElement("div");
	d.appendChild(document.createTextNode(message));
	var h = document.getElementById("history");
	h.appendChild(d);
	window.scrollTo(0, document.body.scrollHeight);
	return d;
}

function make_seen_key(id, text) {
	return id.replace(/@/g, "@@") + "_@_" + text.replace(/@/g, "@@");
}

function receiveMessage(server, time, id, text) {
	var seen_key = make_seen_key(id, text);
	if (!(seen_key in seen)) {
		seen[seen_key] = true;
		rcaddmessagetohistory(text);
		for (i in servers) {
			rcchangeserverstatus(servers[i], "sad");
		}
	}
	rcchangeserverstatus(server, "happy");
}

function receiveMessageEvent(event)  
{  
	for (i in servers) {
		if (event.origin === rcserverbase(servers[i])) {
			messages = JSON.parse(event.data);
			for (j in messages) {
				if ('Time' in messages[j] &&
				    'ID'   in messages[j] &&
				    'Text' in messages[j]) {
					receiveMessage(servers[i], messages[j]['Time'], messages[j]['ID'], messages[j]['Text']);
				}
			}
		}
	}
}

function rcconnect() {
	window.addEventListener("message", receiveMessageEvent, false);  
	for (i in servers) {
		// Create a hidden iframe for same-origin workaround
		var iframe = document.createElement("iframe");
		iframe.setAttribute("src", rcserverbase(servers[i]) + "/frame");
		document.body.insertBefore(iframe, document.body.firstChild);
		// Status bar entry
		var status_indicator = document.createElement("span");
		status_indicator.appendChild(document.createTextNode(servers[i]));
		status_indicator.setAttribute("class", "sad");
		document.getElementById("status").appendChild(status_indicator);
	}
}

function rcsend(d, message) {
	var id = new Date().getTime() + "-" + session + "-" + Math.random();
	seen[make_seen_key(id, message)] = true;
	var path = "/speak" +
		"?id=" + encodeURIComponent(id) +
		"&text=" + encodeURIComponent(message);
	for (i in servers) {
		var uri = rcserverbase(servers[i]) + path;
		var img = document.createElement("img");
		img.setAttribute("src", uri);
		d.appendChild(img);
	}
}

function rckeydown(event) {
	if (event.keyCode == 13) {
		var d = rcaddmessagetohistory(document.input.say.value);
		rcsend(d, document.input.say.value);
		document.input.say.value = "";
		return false;
	}
}
