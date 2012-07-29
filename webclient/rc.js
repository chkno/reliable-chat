var servers = ['chkno.net', 'localhost']

var session = Math.random();

function addport(server) {
	// Add the default port if server doesn't contain a port number already
	if (server.indexOf(":") == -1) {
		return server + ":21059";
	} else {
		return server;
	}
}

function rcsend(d, message) {
	var id = new Date().getTime() + "-" + session + "-" + Math.random();
	var path = "/speak" +
		"?id=" + encodeURIComponent(id) +
		"&text=" + encodeURIComponent(message);
	for (i in servers) {
		var uri = "http://" + addport(servers[i]) + path;
		var img = document.createElement("img");
		img.setAttribute("src", uri);
		d.appendChild(img);
	}
}

function rckeydown(event) {
	if (event.keyCode == 13) {
		var d = document.createElement("div");
		d.appendChild(document.createTextNode(document.input.say.value));
		var h = document.getElementById("history");
		h.appendChild(d);
		window.scrollTo(0, document.body.scrollHeight);
		rcsend(d, document.input.say.value);
		document.input.say.value = "";
		return false;
	}
}
