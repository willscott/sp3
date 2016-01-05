function hex2ip(hex) {
  var ip = '';
  for (var i = 0; i < hex.length - 1; i += 2) {
    ip = ip + (ip.length?'.':'') + parseInt(hex.substr(i, 2), 16);
  }
  return ip;
}
function ip2hex(ip) {
  return ip.split('.').map(function (dec) {
    var hex = parseInt(dec, 10).toString(16);
    if (hex.length < 2) {
      hex = '0' + hex;
    }
    return hex;
  }).join('');
}
function hex2port(hex) {
  var val = parseInt(hex, 16);
  return val;
}
function port2hex(port) {
  var val = parseInt(port, 10).toString(16);
  while (val.length < 4) {
    val = '0' + val;
  }
  return val;
}
function hex2ascii(hex) {
  var txt = '';
  for (var i = 0; i < hex.length - 1; i += 2) {
    txt = txt + String.fromCharCode(parseInt(hex.substr(i, 2), 16));
  }
  return txt;
}
