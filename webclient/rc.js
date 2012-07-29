function rckeydown(event) {
	if (event.keyCode == 13) {
		var d = document.createElement("div");
		d.appendChild(document.createTextNode(document.input.say.value));
		var h = document.getElementById("history");
		h.appendChild(d);
		window.scrollTo(0, document.body.scrollHeight);
		document.input.say.value = "";
		return false;
	}
}
