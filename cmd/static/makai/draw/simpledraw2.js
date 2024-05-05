//Carlos Sanchez - 2017
//randomouscrap98@aol.com
//This script requires randomous.js

function SimpleDraw()
{
   this.drawer = false;
   this.canvas = false;
   this.canvasContainer = false;
   this.easel = false;
   this.info = false;
   this.container = false;

   this._generated = false;
   this._ctrlTool = false;
   this._eTool = false;
   this._toolRadios = false;
   this._cursorPosition = false;

   this._showInfo = true;
   this.fixToolStylingCallback = false;

   this.x = 0;
   this.y = 0;
   this.zoom = 0;
   this.minZoom = 0;
   this.maxZoom = 7;
}

SimpleDraw.StyleID = "simpledraw_autogeneratedsytles";
SimpleDraw.EaselClass = "easel";
SimpleDraw.ContainerClass = "simpledraw";
SimpleDraw.ControlsClass = "controls";
SimpleDraw.ToolboxClass = "toolbox";
SimpleDraw.ToolOptionsClass = "tooloptions";
SimpleDraw.DrawToolsClass = "drawtools";
SimpleDraw.ActionsClass = "actions";
SimpleDraw.LayersClass = "layers";
SimpleDraw.LinksClass = "links";
SimpleDraw.InfoClass = "info";
SimpleDraw.CursorPositionClass = "cursorposition";
SimpleDraw.LayerControlsClass = "layercontrols";
SimpleDraw.CanvasClass = "draw";
SimpleDraw.CanvasContainerClass = "drawcontainer";
SimpleDraw.GridClass = "grid";
SimpleDraw.ColorPickerClass = "colorpicker";
SimpleDraw.WidthPickerClass = "widthpicker";
SimpleDraw.AvailableTools = 
[ 
   "⛶", "eraser", "Eraser",
   "✏", "freehand", "Freehand",
   "✒", "slow", "Slow/Smooth",
   "⚟", "spray", "Spray",
   "▬", "line", "Straight Line",
   "◻", "square", "Square Outline",
   "◩", "fill", "Bucket Fill",
   "◼", "clear", "Full Fill",
   "⚗", "dropper", "Color Select",
   "⤨", "mover", "Layer Mover"
];

SimpleDraw.prototype.SetInfoDisplayed = function(displayed)
{
   this._showInfo = displayed ? true : false;
   console.debug("SetInfoDisplayed called with " + this._showInfo);
   if(this._showInfo)
      this.info.style.display = "";
   else
      this.info.style.display = "none";
};

SimpleDraw.prototype.GetControlsBox = function()
{
   return this.container.querySelector("." + SimpleDraw.ControlsClass); 
};

SimpleDraw.prototype.GetLayerBox = function() 
{ 
   return this.container.querySelector("." + SimpleDraw.LayersClass); 
};

SimpleDraw.prototype.CreateToolButton = function(text, tool, title)
{
   return this._toolRadios.CreateRadioButton(text, tool);
};

SimpleDraw.prototype.CreateRegularButton = function(text, onClick, title)
{
   var button = HTMLUtilities.CreateUnsubmittableButton(text);
   if(title) button.setAttribute("title", title);
   button.addEventListener("click", onClick);
   return button;
};

SimpleDraw.prototype.CreateLayer = function(layerNum)
{
   var lCanvas = document.createElement("canvas");
   lCanvas.setAttribute("data-layer", String(layerNum));
   lCanvas.width = this.canvas.width;
   lCanvas.height = this.canvas.height;
   lCanvas.addEventListener("click", this.SelectLayer.bind(this, lCanvas));
   if(layerNum === 0) CanvasUtilities.Clear(lCanvas, "#FFFFFF");
   return lCanvas;
};

SimpleDraw.prototype.SwapLayers = function(index1, index2)
{
   var firstLayer = this.drawer.buffers[index1];
   var secondLayer = this.drawer.buffers[index2];
   var canvases = this.GetLayerBox().querySelectorAll("canvas");
   this.drawer.buffers[index1] = secondLayer;
   this.drawer.buffers[index2] = firstLayer;
   HTMLUtilities.SwapElements(canvases[index1].parentNode, canvases[index2].parentNode);
   this.drawer.Redraw();
};

//Select the tool represented by the given button. Also updates the UI
SimpleDraw.prototype.SelectTool = function(toolButton) 
{
   this._toolRadios.SelectRadio(toolButton);
};

//Select the layer represented by the given layer element (a canvas). Also
//updates the UI
SimpleDraw.prototype.SelectLayer = function(layer)
{
   var layerNum = layer.getAttribute("data-layer");

   if(!layerNum)
   {
      console.log("Could not select layer using canvas! There is no data-layer attribute!");
      return;
   }

   this.drawer.currentLayer = Number(layerNum);
   var layerBox = HTMLUtilities.FindParentWithClass(layer, SimpleDraw.LayersClass);

   if(!layerBox)
   {
      console.log("Could not find the layer parent while selecting!");
      return;
   }

   HTMLUtilities.SimulateRadioSelect(layer.parentNode, layerBox, "data-selected");

   var layerOpacity = document.querySelector("." + SimpleDraw.LayerControlsClass + ' input[type="range"]');
   if(layerOpacity) layerOpacity.value = this.drawer.GetCurrentLayer().opacity;
};

SimpleDraw.prototype.FixToolStyling = function()
{
   if(this.fixToolStylingCallback)
      this.fixToolStylingCallback();
};

SimpleDraw.prototype.TrySetDefaultStyles = function()
{
   if(document.getElementById(SimpleDraw.StyleID))
      return;

   console.log("Setting up SimpleDraw default styles for the first time");
   var mStyle = StyleUtilities.CreateStyleElement(SimpleDraw.StyleID);
   StyleUtilities.InsertStylesAtTop([mStyle]);

   //General Styling
   mStyle.Append(["." + SimpleDraw.ContainerClass + " *", "." + SimpleDraw.ContainerClass],
      ["padding:0", "margin:0", "display:inline"]);

   //Container styling.
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.EaselClass], SimpleDraw.ContainerClass],
      ["max-width: 100%","max-height:100%","min-width: 100%","min-height: 100%","width: 200px",
       "height: 200px","position:relative","display:block","overflow:hidden"]);
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.EaselClass]], 
      ["background-color:#EEE","z-index:1"]);
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.CanvasContainerClass]], 
      [ "position:absolute","display:block","font-size:0"]);

   //Canvas/grid Styling
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.CanvasClass]],
      StyleUtilities.NoImageInterpolationRules());
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.GridClass]],
      ["width: 100%","height:100%","display:block","position:absolute","top:0","left:0"]);

   //Controls Styling
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.ControlsClass]], 
      ["vertical-align:top","position:absolute","top:0","left:0","z-index:2","height:0"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ControlsClass, "button"]], 
      ["width:30px","height:30px","line-height:16px","vertical-align:top","font-size:24px","display:inline"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ControlsClass, 'input']], 
      ["width:40px","height:30px","font-size:20px","vertical-align:top"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ControlsClass, "canvas"]],
      ["width:28px","height:28px","padding:0","border:1px solid #777"]);

   //Tools styling
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ToolboxClass, "button"]], 
      ["background-color:#EFEFEE","border:1px solid #888"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.WidthPickerClass, 'button']], 
      ["width:15px"]);

   //Action styling
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ActionsClass, "button"],
      ["." + SimpleDraw.ContainerClass, "." + SimpleDraw.WidthPickerClass, "button"]], 
      ["background-color:#E5E4E7","border:1px solid #888"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.ActionsClass, "button:hover:enabled"], 
      ["." + SimpleDraw.ContainerClass, "." + SimpleDraw.WidthPickerClass, "button:hover:enabled"]], 
      ["background-color:#B1E3ff","border:1px solid #888"]);

   //Layers styling
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.LayersClass]], 
      ["display:inline-block"]);
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.LayerControlsClass]], 
      ["text-align: center","display:block"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.LayerControlsClass, "button"]], 
      ["width:20px", "height:20px", "line-height:0", "font-size: 15px","font-weight:bold"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.LayerControlsClass, 'input']], 
      ["width:80px","height:20px"]);

   //Info styling
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.InfoClass]], 
      ["vertical-align:top","position:absolute","bottom:0","left:0","right:0","margin:auto",
       "font-family:monospace","font-size: 12px","z-index:2","background-color:#F4F4F4",
       "padding:2px 4px","width:75px","text-align:center"]);

   //Links styling
   mStyle.AppendClasses([[SimpleDraw.ContainerClass, SimpleDraw.LinksClass]], 
      ["vertical-align:top","position:absolute","bottom:0","left:0","font-family:monospace",
       "font-size: 12px","z-index:2"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "." + SimpleDraw.LinksClass, "a"]],
      ["padding:3px","display:inline-block","background-color:#EEE",
       "margin-right:3px"]);

   //State styling.
   mStyle.Append([["." + SimpleDraw.ContainerClass, "button:disabled"]], 
      ["background-color:#D7D7D7"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "button[data-selected]"]], 
      ["background-color: #91d3ff","border-color:#89A"]);
   mStyle.Append([["." + SimpleDraw.ContainerClass, "[data-selected]", "canvas"]], 
      ["border:2px solid blue", "width:26px","height:26px"]);
};

SimpleDraw.prototype.Generate = function(width, height, layerCount, maxUndos)
{
   if(this._generated)
   {
      console.log("Tried to generate SimpleDraw again");
      throw "This SimpleDraw is already generated!";
   }

   width = width || 200;
   height = height || 200;
   layerCount = layerCount || 4;
   maxUndos = maxUndos || 10;
   console.debug("Generating SimpleDraw, w: " + width + " h: " + height + 
      ", l: " + layerCount + ", u: " + maxUndos);

   var me = this;

   //----Canvas and Easel setup----
   //The easel holds the canvas and lets you move it around and whatever.
   var easel = document.createElement("div");
   var canvas = document.createElement("canvas");
   var canvasContainer = document.createElement("div");
   //var grid = document.createElement("div");
   easel.className = SimpleDraw.EaselClass;
   canvas.className = SimpleDraw.CanvasClass;
   canvasContainer.className = SimpleDraw.CanvasContainerClass;
   //grid.className = SimpleDraw.GridClass;
   canvas.width = width;
   canvas.height = height;
   this.canvas = canvas;
   this.canvasContainer = canvasContainer;
   this.easel = easel;
   canvasContainer.appendChild(canvas);
   //canvasContainer.appendChild(grid);
   easel.appendChild(canvasContainer); //Now the easel and canvas are all set up.
   easel.addEventListener("contextmenu", function(e) { e.preventDefault(); });

   //----Tool setup----
   var tools = document.createElement("div");
   var toolHolder = document.createElement("div");
   var toolOptions = document.createElement("div");
   tools.className = SimpleDraw.ToolboxClass;
   toolOptions.className = SimpleDraw.ToolOptionsClass;
   toolHolder.className = SimpleDraw.DrawToolsClass;

   var colorPicker = document.createElement("input"); //The first item in the toolbox is the color selector
   colorPicker.className = SimpleDraw.ColorPickerClass;
   colorPicker.setAttribute("type", "color");
   colorPicker.setAttribute("title", "Tool Color");
   colorPicker.addEventListener("change", function(e) { me.drawer.color = colorPicker.value; });
   toolOptions.appendChild(colorPicker);

   //2023: Second element is HSV slider to manually modify color input
   var hsvPicker = document.createElement("button");
   hsvPicker.textContent = "COL";
   hsvPicker.addEventListener('click', function(e) {
      var newpicker = me.CreateColorPicker(colorPicker);
      UXUtilities.Alert(newpicker);
   });
   toolOptions.appendChild(hsvPicker);

   var widthContainer = document.createElement("div");
   var widthDown = document.createElement("button");
   var widthUp = document.createElement("button");
   var widthPicker = document.createElement("input"); //Immediately after the color selector is the width selector.
   widthContainer.className = SimpleDraw.WidthPickerClass;
   widthDown.innerHTML = "<";
   widthUp.innerHTML = ">";
   widthPicker.setAttribute("title", "Tool Width");
   var updateWidth = function()
   {
      var newWidth = MathUtilities.MinMax(Number(widthPicker.value), 1, 99);
      if(!isNaN(newWidth)) me.drawer.lineWidth = newWidth;
      widthPicker.value = me.drawer.lineWidth;
   };
   widthPicker.addEventListener("change", updateWidth);
   widthDown.addEventListener("click", function()
   {
      widthPicker.value = Number(widthPicker.value) - 1;
      updateWidth();
   });
   widthUp.addEventListener("click", function()
   {
      widthPicker.value = Number(widthPicker.value) + 1;
      updateWidth();
   });
   widthContainer.appendChild(widthDown);
   widthContainer.appendChild(widthPicker);
   widthContainer.appendChild(widthUp);
   toolOptions.appendChild(widthContainer);

   var shapeRadios = new RadioSimulator(toolOptions, "data-shape", 
      function(shape) { me.drawer.lineShape = shape; }, true);
   toolOptions.appendChild(shapeRadios.CreateRadioButton("●", "hardcircle"));
   toolOptions.appendChild(shapeRadios.CreateRadioButton("■", "hardsquare"));
   toolOptions.appendChild(shapeRadios.CreateRadioButton("◉", "normalcircle"));
   toolOptions.appendChild(shapeRadios.CreateRadioButton("▣", "normalsquare"));

   tools.appendChild(toolOptions);

   this._toolRadios = new RadioSimulator(toolHolder, "data-tool",
      function(toolString) {me.drawer.currentTool = toolString;});

   for(i = 0; i < SimpleDraw.AvailableTools.length; i+=3)
   {
      toolHolder.appendChild(this.CreateToolButton(SimpleDraw.AvailableTools[i],
         SimpleDraw.AvailableTools[i + 1], SimpleDraw.AvailableTools[i + 2]));
   }

   tools.appendChild(toolHolder);

   var fullClearButton = this.CreateRegularButton("✖", function() { me.drawer.ClearLayer();}, "Layer Clear");
   tools.appendChild(fullClearButton); //TODO: the last tool is ALWAYS the fullclear button. This may be bad?

   //----Actions setup----
   var actions = document.createElement("div");
   actions.className = SimpleDraw.ActionsClass;

   var undoButton = this.CreateRegularButton("↶", 
      function() { me.drawer.Undo(); }, "Undo");
   var redoButton = this.CreateRegularButton("↷", 
      function() { me.drawer.Redo(); }, "Redo");
   var scaleDownButton = this.CreateRegularButton("-", 
      function() { me.UpdateZoom(-1); }, "Zoom Out");
   var scaleUpButton = this.CreateRegularButton("+", 
      function() { me.UpdateZoom(1); }, "Zoom In");
   var recenterButton = this.CreateRegularButton("👁", 
      function() { me.ResetNavigation(); }, "Reset Zoom");
   var fullscreenButton = this.CreateRegularButton("◰", 
      function() 
      {
         if(ScreenUtilities.IsFullscreen())
            ScreenUtilities.ExitFullscreen();
         else
            ScreenUtilities.LaunchIntoFullscreen(me.container);
      }, "Fullscreen Toggle");

   undoButton.dataset.action = "undo";
   redoButton.dataset.action = "redo";
   scaleDownButton.dataset.action = "scaledown";
   scaleUpButton.dataset.action = "scaleup";
   recenterButton.dataset.action = "recenter";
   fullscreenButton.dataset.action = "fullscreen";

   actions.appendChild(undoButton);
   actions.appendChild(redoButton);
   actions.appendChild(scaleDownButton);
   actions.appendChild(scaleUpButton);
   actions.appendChild(recenterButton);
   actions.appendChild(fullscreenButton);

   //----Layers setup----
   var layerBox = document.createElement("div");
   var layerControlsBox = document.createElement("div");
   layerBox.className = SimpleDraw.LayersClass;
   layerControlsBox.className = SimpleDraw.LayerControlsClass;

   var layers = [];
   for(i = 0; i < layerCount; i++)
   {
      var lCanvas = this.CreateLayer(i);
      var lContainer = document.createElement("div");
      lContainer.className = "layer";
      lCanvas.setAttribute("title", "Layer " + (i + 1));
      layers.push(lCanvas);
      lContainer.appendChild(lCanvas);
      layerBox.appendChild(lContainer);
   }

   var shiftLayerLeft = this.CreateRegularButton("<", function() 
   { 
      var currentLayer = me.drawer.CurrentLayerIndex();
      me.SwapLayers(currentLayer, (currentLayer - 1 + layerCount) % layerCount); 
   }, "Shift Layer Left");
   var shiftLayerRight = this.CreateRegularButton(">", function() 
   { 
      var currentLayer = me.drawer.CurrentLayerIndex();
      me.SwapLayers(currentLayer, (currentLayer + 1) % layerCount); 
   }, "Shift Layer Right");
   var layerOpacity = document.createElement("input");
   layerOpacity.setAttribute("type", "range");
   layerOpacity.setAttribute("min", "0");
   layerOpacity.setAttribute("max", "1");
   layerOpacity.setAttribute("step", "0.05");
   layerOpacity.setAttribute("title", "Current Layer Opacity");
   layerOpacity.addEventListener("input", function(e) 
   { 
      me.drawer.GetCurrentLayer().opacity = layerOpacity.value;
      me.drawer.Redraw();
   });

   layerControlsBox.appendChild(shiftLayerLeft);
   layerControlsBox.appendChild(layerOpacity);
   layerControlsBox.appendChild(shiftLayerRight);
   layerBox.appendChild(layerControlsBox);

   //----Links Setup----
   var links = document.createElement("div");
   links.className = SimpleDraw.LinksClass;

   var download = document.createElement("a"); //The very last item in the controls is the download thingy
   download.href = "#";
   download.innerHTML = "Download";
   download.className = "download";
   download.addEventListener("click", function(e)
   {
      download.href = me.canvas.toDataURL();
      download.download = "draw_" + (Math.floor(new Date().getTime()/1000)) + ".png";
   }, false);
   links.appendChild(download);
   
   var help = document.createElement("a");
   help.href = "#";
   help.innerHTML = "Help";
   help.className = "help";
   help.addEventListener("click", function(e)
   {
      var helpText = "A simple drawing application with 4 layers and simple tools." +
         "\n\n" +
         "E : toggle eraser\n" +
         "Z : undo\n" +
         "Y : redo\n" +
         "M : increase brush size\n" +
         "N : decrease brush size\n" +
         "+ : zoom in\n" +
         "- : zoom out\n" +
         "CTRL (hold) : dropper";
      UXUtilities.Alert(helpText);//alert(helpText);
   }, false);
   links.appendChild(help);

   //----Info setup----
   var infoElement = document.createElement("div");
   var cursorPositionElement = document.createElement("span");
   infoElement.className = SimpleDraw.InfoClass;
   cursorPositionElement.className = SimpleDraw.CursorPositionClass;
   this.info = infoElement;
   this._cursorPosition = cursorPositionElement;
   infoElement.appendChild(cursorPositionElement);

   //----Container setup----
   var container = document.createElement("div");
   var controls = document.createElement("div");
   container.className = SimpleDraw.ContainerClass;
   controls.className = SimpleDraw.ControlsClass;
   this.container = container;

   controls.appendChild(tools);
   controls.appendChild(actions);
   controls.appendChild(layerBox);
   container.appendChild(controls);
   container.appendChild(easel);
   container.appendChild(infoElement);
   container.appendChild(links);

   //**----Drawer setup----**
   this.drawer = new CanvasDrawer();
   this.drawer.WheelZoom = 1;
   this.drawer.Attach(canvas, layers, maxUndos);
   this.drawer.OnUndoStateChange = function() 
   { 
      undoButton.disabled = !me.drawer.CanUndo();
      redoButton.disabled = !me.drawer.CanRedo();
   };
   this.drawer.OnLayerChange = function(layer)
   {
      me.SelectLayer(layers[layer]);
   };
   this.drawer.OnColorChange = function(color)
   {
      colorPicker.value = StyleUtilities.GetColor(color).ToHexString();
   };

   var oldOnAction = this.drawer.OnAction;
   this.drawer.OnAction = function(data, context)
   {
      if(me._showInfo) 
      {
         me._cursorPosition.innerHTML = Math.floor(data.x) + "," + Math.floor(data.y);
      }
      if(data.action & CursorActions.Pan)
      {
         if(!(data.action & CursorActions.Start))
         {
            me.x += data.realX - me.oldX;
            me.y += data.realY - me.oldY;
            me.RefreshLocation();
         }
         me.oldX = data.realX;
         me.oldY = data.realY;
      }
      if(data.action & CursorActions.Zoom)
      {
         me.UpdateZoom(data.zoomDelta, data.realX, data.realY);
      }
      oldOnAction(data, context);
   };

   this.drawer.Redraw();
   this.drawer.DoUndoStateChange();

   colorPicker.value = this.drawer.color;
   widthPicker.value = this.drawer.lineWidth;
   this.SelectTool("freehand"); 
   shapeRadios.SelectRadio("hardcircle");
   this.SelectLayer(layers[1]);
   this.FixToolStyling();

   //Insert the default styles into the page (at the beginning, I hope) so that
   //other people can override them.
   this.TrySetDefaultStyles();

   //NOTE: These events are permanently affixed to the document since there's
   //no way to "clean up" the simple draw. This is probably a bad idea, but
   //let's hope that for now, this won't be an issue.
   this.container.addEventListener("keydown", function(e)
   {
      if(!me._ctrlTool && e.ctrlKey)
      {
         me._ctrlTool = me.drawer.currentTool;
         me.SelectTool("dropper");
      }
   });
   this.container.addEventListener("keyup", function(e)
   {
      if(me._ctrlTool)
      {
         if(me._ctrlTool === "eraser") me._ctrlTool = "freehand";
         me.SelectTool(me._ctrlTool);
         me._ctrlTool = false;
      }
   });
   this.container.addEventListener("keypress", function(e)
   {
      if(!HTMLUtilities.FindParentWithClass(e.target, SimpleDraw.ControlsClass) && 
         e.target.tagName === "INPUT") 
      {
         return;
      }
         //|| e.target.tagName === "TEXTAREA" || e.target.tagName === "") return;

      var ch = String.fromCharCode(e.charCode);
      console.debug("Doing simpledraw keypress: " + ch);

      if(ch === "+")
      {
         me.UpdateZoom(1);
      }
      else if(ch === "-")
      {
         me.UpdateZoom(-1);
      }
      else if (ch === "z")
      {
         me.drawer.Undo();
      }
      else if (ch === "y")
      {
         me.drawer.Redo();
      }
      else if (ch === "m")
      {
         if(widthPicker.value < widthPicker.max)
         {
            widthPicker.value++;
            me.drawer.lineWidth = widthPicker.value;
         }
      }
      else if (ch === "n")
      {
         if(widthPicker.value > widthPicker.min)
         {
            widthPicker.value--;
            me.drawer.lineWidth = widthPicker.value;
         }
      }
      else if(ch === "e")
      {
         if(me._eTool && me.drawer.currentTool === "eraser")
         {
            me.SelectTool(me._eTool);
            me._eTool = false;
         }
         else if(me.drawer.currentTool !== "eraser")
         {
            me._eTool = me.drawer.currentTool;
            me.SelectTool("eraser");
         }
         else
         {
            me._eTool = false;
         }
      }
      else
      {
         return;
      }

      e.preventDefault();
   });

   this.container.setAttribute("tabindex", "-1");
   window.addEventListener("resize", function() { me.RefreshLocation(); me.FixToolStyling(); });
   window.addEventListener("message", function(e)
   {
      if(e.data.getImage)
      {
         e.source.postMessage({image:me.drawer._canvas.toDataURL("image/png"),type:"getImage"}, "*");
      }
      else if(e.data.recenter)
      {
         console.debug("An outside member requested a recenter");
         me.ResetNavigation();
      }
   });

   this._generated = true;
   return container;
};

SimpleDraw.prototype.UpdateZoom = function(delta, cx, cy)
{
   delta = delta || 0;
   var newZoom = this.zoom + delta;

   console.trace("Attempting to update zoom. cx: " + cx + ", cy: " +
      cy + ", newZoom: " + newZoom);

   if(newZoom >= this.minZoom && newZoom <= this.maxZoom)
   {
      console.debug("Zoom allowed. Setting zoom to " + newZoom);
      var easelDim = StyleUtilities.GetTrueRect(this.easel);
      cx = (cx === undefined) ? 0 : cx - easelDim.width / 2;
      cy = (cy === undefined) ? 0 : cy - easelDim.height / 2;
      var oldDim = StyleUtilities.GetTrueRect(this.canvas);
      this.zoom = newZoom;
      CanvasUtilities.SetScaling(this.canvas, Math.floor(Math.pow(2, newZoom)));//, this.canvasContainer);
      var newDim = StyleUtilities.GetTrueRect(this.canvas);
      this.canvasContainer.style.width = newDim.width + "px";
      this.canvasContainer.style.height = newDim.height + "px";
      this.x = (newDim.width / oldDim.width) * (this.x - cx) + cx;
      this.y = (newDim.height / oldDim.height) * (this.y - cy) + cy;
      this.RefreshLocation();
      this.FixToolStyling();
   }
};

SimpleDraw.prototype.RefreshLocation = function()
{
   var rect = StyleUtilities.GetTrueRect(this.canvas);
   var cont = StyleUtilities.GetTrueRect(this.easel);
   //console.log(rect);
   console.trace("Refreshing location. x: " + this.x + ", y: " + this.y +
      ", easelw: " + rect.width + ", easelh: " + rect.h);
   this.canvasContainer.style.top = (cont.height - rect.height) / 2 + this.y + "px";
   this.canvasContainer.style.left = (cont.width - rect.width) / 2 + this.x + "px";
};

SimpleDraw.prototype.ResetNavigation = function()
{
   console.debug("Resetting navigation.");
   this.x = 0;
   this.y = 0;
   this.zoom = this.minZoom;
   this.RefreshLocation();
   this.UpdateZoom(); //SetCanvasZoom(this.zoom);
};

SimpleDraw.prototype.CreateColorPicker = function(input)
{
   var coldiv = document.createElement("div");
   coldiv.style.width = "2em";
   coldiv.style.height = "2em";
   coldiv.style.margin = "auto";

   var rgb = StyleUtilities.GetColor(input.value); //Color type
   var hsv = SimpleDraw.RgbToHsv(rgb.r, rgb.g, rgb.b);
   //console.log(rgb, hsv);

   var h = document.createElement("input");
   h.type = "range";
   h.min = 0;
   h.max = 1;
   h.step = 0.01;
   h.value = hsv.h;
   var s = document.createElement("input");
   s.type = "range";
   s.min = 0;
   s.max = 1;
   s.step = 0.01;
   s.value = hsv.s;
   var v = document.createElement("input");
   v.type = "range";
   v.min = 0;
   v.max = 1;
   v.step = 0.01;
   v.value = hsv.v;

   var updateCol = function()
   {
      var newRgb = SimpleDraw.HsvToRgb(h.value, s.value, v.value);
      var newRgbCol = new Color(newRgb.r, newRgb.g, newRgb.b);
      var colString = newRgbCol.ToHexString();
      //console.log("Changing: ", newRgb, newRgbCol, colString);
      coldiv.style.backgroundColor = colString;
      input.value = colString;
   };

   h.oninput = updateCol;
   s.oninput = updateCol;
   v.oninput = updateCol;

   //Make sure the coldiv has the right color or something
   updateCol();

   var ht = document.createElement("div");
   ht.textContent = "Hue:";
   var st = document.createElement("div");
   st.textContent = "Saturation:";
   var vt = document.createElement("div");
   vt.textContent = "Value:";

   var container = document.createElement("div");
   container.appendChild(coldiv);
   container.appendChild(ht);
   container.appendChild(h);
   container.appendChild(st);
   container.appendChild(s);
   container.appendChild(vt);
   container.appendChild(v);

   //container.style.
   //container.style.display = "flex";
   //container.style.flexDirection = "column";

   return container;
};

//These two functions taken from https://stackoverflow.com/a/17243070/1066474

/* accepts parameters
 * h  Object = {h:x, s:y, v:z}
 * OR 
 * h, s, v
*/
SimpleDraw.HsvToRgb = function (h, s, v)
{
   var r, g, b, i, f, p, q, t;
   if (arguments.length === 1) {
       s = h.s, v = h.v, h = h.h;
   }
   i = Math.floor(h * 6);
   f = h * 6 - i;
   p = v * (1 - s);
   q = v * (1 - f * s);
   t = v * (1 - (1 - f) * s);
   switch (i % 6) {
       case 0: r = v, g = t, b = p; break;
       case 1: r = q, g = v, b = p; break;
       case 2: r = p, g = v, b = t; break;
       case 3: r = p, g = q, b = v; break;
       case 4: r = t, g = p, b = v; break;
       case 5: r = v, g = p, b = q; break;
   }
   return {
       r: Math.round(r * 255),
       g: Math.round(g * 255),
       b: Math.round(b * 255)
   };
};

/* accepts parameters
 * r  Object = {r:x, g:y, b:z}
 * OR 
 * r, g, b
*/
SimpleDraw.RgbToHsv = function (r, g, b) {
   if (arguments.length === 1) {
       g = r.g, b = r.b, r = r.r;
   }
   var max = Math.max(r, g, b), min = Math.min(r, g, b),
       d = max - min,
       h,
       s = (max === 0 ? 0 : d / max),
       v = max / 255;

   switch (max) {
       case min: h = 0; break;
       case r: h = (g - b) + d * (g < b ? 6: 0); h /= 6 * d; break;
       case g: h = (b - r) + d * 2; h /= 6 * d; break;
       case b: h = (r - g) + d * 4; h /= 6 * d; break;
   }

   return {
       h: h,
       s: s,
       v: v
   };
}