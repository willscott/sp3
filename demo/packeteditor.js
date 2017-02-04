// small hex-editor to display a packet, along with handling of annotations
// to allow direct twiddling of fields.

var PacketField = function (label, offset, length, toText, toHex) {
  this.label = label;
  this.offset = offset;
  this.length = length;
  this.toText = toText;
  this.toHex = toHex;
  this.background = '#ffd';
};

PacketField.prototype.render = function (into) {
  into.style.display = 'inline-block';
  into.style.position = 'relative';
  into.style.background = this.background;
  var val = document.createElement('span');
  val.innerHTML = this.value;
  into.appendChild(val);
  this.valel = val;
  var label = document.createElement('span');
  label.style.position = 'absolute';
  label.style.left = 0;
  label.style.top = '-0.7em';
  label.style.fontSize = '0.40em';
  label.innerHTML = this.label;
  into.appendChild(label);
  var input = document.createElement('input');
  input.value = this.toText(this.value);
  input.style.display = 'none';
  input.style.border = '1px solid black';
  input.style.width = this.length + 'em';
  input.addEventListener('change', function() {
    this.value = this.toHex(input.value);
    this.onChange();
  }.bind(this), true);
  input.addEventListener('blur', function(input, val) {
    input.style.display = 'none';
    val.style.display = 'inline';
    val.innerHTML = this.value;
  }.bind(this, input, val), true);
  into.appendChild(input);
  this.inputel = input;

  val.addEventListener('click', function (input) {
    this.style.display = 'none';
    input.style.display = 'block';
    input.focus();
  }.bind(val, input), true);
};

PacketField.prototype.set = function (value) {
  this.value = value;
  if (this.valel) {
    this.valel.innerHTML = value;
    this.inputel.value = this.toText(this.value);
    this.onChange();
  }
};

PacketField.prototype.onChange = function () {};

var VarLenField = function (label, offset, toText, toHex) {
  PacketField.call(this, label, offset, 0, toText, toHex);
};

VarLenField.prototype.toEdit = function () {
  this.lines.forEach(function(line) {
    if (line.el) {
      line.el.style.display = 'none';
    } else {
      line.style.display = 'none';
    }
  });
  this.input.style.display = 'block';
  this.input.focus();
};

VarLenField.prototype.render = function (into) {
  into.style.display = 'inline-block';
  into.style.position = 'relative';
  into.style.height = '1em';
  var label = document.createElement('span');
  label.style.position = 'absolute';
  label.style.left = 0;
  label.style.top = '-0.7em';
  label.style.fontSize = '0.40em';
  label.innerHTML = this.label;
  into.appendChild(label);
  var input = document.createElement('textarea');
  input.value = this.toText(this.value);
  input.style.display = 'none';
  input.style.border = '1px solid black';
  input.style.width = '100%';
  input.addEventListener('change', function() {
    this.value = this.toHex(input.value);
    this.onChange();
  }.bind(this), true);
  input.addEventListener('blur', function(input) {
    input.style.display = 'none';
    this.lines.forEach(function (line) {
      if (line.el) {
        line.el.style.display = 'block';
      } else {
        line.style.display = 'inline-block';
      }
    });
    //val.style.display = 'inline';
    //val.innerHTML = this.value;
  }.bind(this, input), true);
  into.appendChild(input);
  this.input = input;
  setTimeout(function () {
    this.lines[0] = this.lines[0].el.lastChild;
    this.lines.forEach(function (line) {
      if (line.el) {
        line.el.addEventListener('click', this.toEdit.bind(this));
      } else {
        line.addEventListener('click', this.toEdit.bind(this));
      }
    }.bind(this));
  }.bind(this), 0);
};

var ComputedField = function (label, offset, length, compute) {
  this.label = label;
  this.offset = offset;
  this.length = length;
  this.compute = compute;
  this.background = '#fcf';
};

ComputedField.prototype.render = function (into) {
  into.style.display = 'inline-block';
  into.style.position = 'relative';
  into.style.background = this.background;
  var val = document.createElement('span');
  val.innerHTML = this.value;
  this.el = val;
  into.appendChild(val);
  var label = document.createElement('span');
  label.style.position = 'absolute';
  label.style.left = 0;
  label.style.top = '-0.7em';
  label.style.fontSize = '0.40em';
  label.innerHTML = this.label;
  into.appendChild(label);
};

ComputedField.prototype.recompute = function (packet) {
  var value = this.compute(packet);
  if (value != this.value) {
    this.value = value;
    this.el.innerHTML = value;
    this.onChange();
  }
};

ComputedField.prototype.onChange = function () {};

var PacketEditor = function (element, fields, width) {
  this.el = element;
  this.el.style.display = 'none';
  this.root = document.createElement('div');
  this.root.style.fontFamily = 'Monospace';
  this.el.parentNode.insertBefore(this.root, this.el);
  this.fields = fields;
  this.width = width;
  this.lines = [];
  this.render();
};

PacketEditor.prototype.recompute = function () {
  if (this.recomputing) {
    return;
  }
  var oldvalue = this.el.value;
  this.recomputing = true;
  for (var i = 0; i < this.fields.length; i += 1) {
    if (this.fields[i].recompute) {
      this.fields[i].recompute(oldvalue);
    }
  }
  this.recomputing = false;
  if (this.el.value != oldvalue) {
    this.recompute();
  }
};

PacketEditor.prototype.render = function () {
  var text = this.el.value;
  // Recalculate lines.
  this.lines = [];
  while (text.length > 0) {
    var line = new PacketEditorLine(text.substr(0, this.width));
    if (text.length > this.width) {
      text = text.substr(this.width);
    } else {
      text = "";
    }
    line.onChange = function(offset, line) {
      var oldValue = this.el.value.split('');
      oldValue.splice(offset, line.value.length, line.value);
      this.el.value = oldValue.join('');
      this.recompute();
    }.bind(this, this.lines.length * this.width, line);
    this.lines.push(line);
  }
  // position fields.
  this.fields.forEach(function (field) {
    var offset = field.offset;
    var line = this.lines[Math.floor(offset / this.width)];
    if (line) {
      line.addField(offset % this.width, field);
    }
    // indefinite fields learn about subsequent lines
    if (!field.length) {
      var lines = [];
      for (var i = Math.floor(offset / this.width); i < this.lines.length; i += 1) {
        lines.push(this.lines[i]);
      }
      field.lines = lines;
      field.value = this.el.value.substr(offset);
    }
  }.bind(this));
  // render output.
  this.lines.forEach(function (line) {
    line.render(this.root);
  }.bind(this));
};

var PacketEditorLine = function (value) {
  this.value = value;
  this.offsets = [];
  this.fields = {};
};

PacketEditorLine.prototype.addField = function (offset, field) {
  this.fields[offset] = field;
  this.offsets.push(offset);
  field.onChange = function (offset, field) {
    var oldvalue = this.value;
    var vals = oldvalue.split('');
    vals.splice(offset, field.length, field.value);
    this.value = vals.join('');
    if (this.value != oldvalue) {
      this.onChange();
    }
  }.bind(this, offset, field);
};

PacketEditorLine.prototype.render = function (into) {
  var el = document.createElement('span');
  el.className = 'editorLine';
  this.el = el;
  var offsets = this.offsets.sort(function(a,b) {return a - b;});
  var i = 0, next;
  while (i < this.value.length) {
      var run = document.createElement('span');
      run.className = 'editorRun';
      next = offsets[0];
      if (next === undefined) {
        run.innerHTML = this.value.substr(i);
        i = this.value.length;
      } else if (next == i) {
        if (this.fields[next].length) {
          this.fields[next].value = this.value.substr(next, this.fields[next].length);
        }
        this.fields[next].render(run);
        i += this.fields[next].length;
        offsets.shift();
      } else {
        run.innerHTML = this.value.substr(i, next - i);
        i = next;
      }
      el.appendChild(run);
  }

  into.appendChild(el);
};

PacketEditorLine.prototype.onChange = function () {};
