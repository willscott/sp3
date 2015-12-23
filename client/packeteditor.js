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
  into.style.display='inline-block';
  into.style.background = this.background;
  var val = document.createElement('span');
  val.innerText = this.value;
  into.appendChild(val);
  var label = document.createElement('span');
  label.innerText = this.label;
  into.appendChild(label);
  var input = document.createElement('input');
  input.value = this.toText(this.value);
  input.addEventListener('change', function() {
    this.value = this.toHex(input.value);
    this.onChange();
  }.bind(this), true);
  into.appendChild(input);
};

PacketField.prototype.onChange = function () {};

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

PacketEditor.prototype.render = function () {
  var text = this.el.value;
  // Recalculate lines.
  this.lines = [];
  while(text.length > 0) {
    var line = new PacketEditorLine(text.substr(0, this.width));
    if (text.length > this.width) {
      text = text.substr(this.width);
    } else {
      text = "";
    }
    line.onChange = function(offset, line) {
      var oldValue = this.el.value;
      oldValue.splice(offset, this.width, line.value);
      this.el.value = oldValue;
    }.bind(this, this.lines.length * this.width, line);
    this.lines.push(line);
  }
  // position fields.
  this.fields.forEach(function (field) {
    var offset = field.offset;
    var line = this.lines[math.floor(offset / this.width)];
    if (line) {
      line.addField(offset % this.width, field);
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
    this.value.splice(offset, field.length, field.value);
    this.onChange();
  }.bind(this, offset, field);
};

PacketEditorLine.prototype.render = function (into) {
  var el = document.createElement('span');
  el.className = 'editorLine';
  var offsets = this.offsets.sort();
  var i = 0, next;
  while (i < this.value.length) {
      var run = document.createElement('span');
      run.className = 'editorRun';
      next = offsets.pop();
      if (next === undefined) {
        run.innerHTML = this.value.substr(i);
        i = this.value.length;
      } else if (next == i) {
        this.fields[next].value = this.value.substr(next, this.fields[next].length);
        this.fields[next].render(run);
        i += this.fields[next].length;
      } else {
        run.innerHTML = this.value.substr(i, next - i);
        i = next;
      }
      el.appendChild(run);
  }

  into.appendChild(el);
};

PacketEditorLine.prototype.onChange = function () {};
