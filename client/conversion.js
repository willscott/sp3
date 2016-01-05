function htons(hex) {
  while (hex.length < 4) {
    hex = '0' + hex;
  }
  return hex[2] + hex[3] + hex[0] + hex[1];
}
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
  var val = parseInt(htons(hex), 16);
  return val;
}
function port2hex(port) {
  var val = parseInt(port, 10).toString(16);
  return htons(val);
}
function hex2ascii(hex) {
  var txt = '';
  for (var i = 0; i < hex.length - 1; i += 2) {
    txt = txt + String.fromCharCode(parseInt(hex.substr(i, 2), 16));
  }
  return txt;
}
function ascii2hex(ascii) {
  return ascii.split('').map(function(char) {
    var hex = char.charCodeAt(0).toString(16);
    if (hex.length < 2) {
      hex = '0' + hex;
    }
    return hex;
  }).join('');
}
function identity(value) {
  return value;
}
function totalLength(packet) {
  return htons(packet.length.toString(16));
}
function ipChecksum(packet) {
  var header = packet.substr(0, 40);
  //checksum is 0 during recalculation
  header[20] = '0';
  header[21] = '0';
  header[22] = '0';
  header[23] = '0';
  return onesComplement(header);
}
function udpLength(packet) {
  return htons((packet.length - 40).toString(16));
}
function udpChecksum(packet) {
  var header =
      packet.substr(24, 8) + // srcip
      packet.substr(32, 8) + // destip
      "00" +
      packet.substr(18, 2) + // protocol
      packet.substr(48, 4) + // length
      packet.substr(40); // udp header + data.
  while (header.length % 4 !== 0) {
    header += '0';
  }
  return onesComplement(header);
}
function onesComplement(hex) {
  var i;
  while (hex.length > 4) {
    var sum = 0;
    for (i = 0; i < hex.length - 1; i += 4) {
      sum += parseInt(hex.substr(i, 4), 16);
    }
    hex = sum.toString(16);
  }
  var binary = parseInt(hex, 16).toString(2);
  while (binary.length < 16) {
    binary = "0" + binary;
  }
  var out = '';
  for (i = 0; i < 16; i += 1) {
    if (binary[i] == "0") {
      out += "1";
    } else {
      out += "0";
    }
  }
  return parseInt(out, 2).toString(16);
}
