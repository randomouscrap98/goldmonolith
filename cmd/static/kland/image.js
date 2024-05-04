window.addEventListener("load", function(event)
{
   var pastediv = document.getElementById("pastediv");
   var slideshowButtons = document.querySelectorAll("[data-slideshow]");
   var slideshowExit = document.getElementById("slideshowexit");

   if(pastediv)
   {
      pastediv.addEventListener("paste", function(e) 
      {
         //Try using "clipboard data" (which is a GOOD API that firefox is too
         //stubborn to implement because it makes life too easy). If that doesn't
         //work, try using 12me's image uploader thing (which assumes the image
         //gets pasted as an element, which again only works on firefox)
         if(!clipboardDataUpload(e, pastediv.dataset.bucket))
         {
            imagePaste12Me21(pastediv, function(image)
            {
               tryUploadImageWithAutomaticCompression(image, pastediv.dataset.bucket); 
            });
         }
      });
   }

   slideshowExit.addEventListener("click", endSlideshow);
   slideshowNext().addEventListener("click", function() { slideshowShift(1); } );
   slideshowBack().addEventListener("click", function() { slideshowShift(-1); } );

   for(var i = 0; i < slideshowButtons.length; i++)
      slideshowButtons[i].addEventListener("click", startSlideshow);
});
function gotoBucket()
{
   var bucket = document.getElementById("gotoThisBucket");
   window.location.href = location.pathname + "?bucket=" + bucket.value;
}
function setIPP()
{
   writeNormalCookie("ipp", Number(document.getElementById("pickIPP").value));            
   location.reload();
}

//A function provided by 12me for ludicrously easy image pasting.
function imagePaste12Me21(element, callback) 
{
   console.log("Running 12me paster on element: ");
   console.log(element);

   var startTime = (new Date().getTime());
   var repeater = setInterval(function()
   {
      var image = element.getElementsByTagName("img")[0];
      var expired = ((new Date().getTime()) - startTime) > 5000;

      //Skip until the image is loaded or we timeout
      if(!image && !expired) return;
      clearInterval(repeater);

      if(expired)
      {
         console.log("Expiring the paste handler. The user probably didn't paste an image");
         return;
      }

      if (image)
      {
         console.log("12me image paster found image. Running callback");
         callback(image);
         image.remove();
      }
      else
      {
         alert("Couldn't find image!");
      }

   }, 100);
}

//Put image on canvas and return canvas. Useful for getting the image as blob.
function drawImageOnCanvas(image)
{
   // var originalWidth = image.style.width;
   // var originalHeight = image.style.height;
   // image.style.width = "unset"; image.style.height = "unset";

   console.log("Drawing image on canvas (probably for upload). Width: " + 
      image.naturalWidth + ", height: " + image.naturalHeight);

   var canvas = document.getElementById("scratchCanvas");
   canvas.width = image.naturalWidth;
   canvas.height = image.naturalHeight;
   canvas.getContext("2d").drawImage(image, 0, 0);

   //image.style.width = originalWidth; image.style.height = originalHeight;
   return canvas;
}

function tryUploadImageWithAutomaticCompression(image, bucket)
{
   console.log("Trying to upload image to bucket: " + bucket);
   var canvas = drawImageOnCanvas(image);
   canvas.toBlob(function(blob)
   {
      if (!blob) {
        alert("Couldn't convert to blob (browser doesn't say why)");
        return;
      } else {
        console.log("Converted image to png blob");
      }
      if(blob.size > 750000)
      {
         console.log("Blob png is too big. Trying jpeg");
         canvas.toBlob(function(blobj)
         {
            uploadBlob(blobj, bucket, "blob.jpg");
         }, "image/jpeg", 0.90);
      }
      else
      {
         uploadBlob(blob, bucket);
      }
   });
}

function clipboardDataUpload(event, bucket)
{
   try
   {
      console.log("doing paste event");

      var clipboardData = event.clipboardData || window.clipboardData;
      if(!clipboardData || !clipboardData.items) { return false; }

      //Only do this if we have the items interface
      var firstItem = clipboardData.items[0];
      var blob = firstItem.getAsFile();
      var image = new Image();

      image.addEventListener("load", function()
      {
         tryUploadImageWithAutomaticCompression(image, bucket);
      });

      image.src = URL.createObjectURL(blob);

      event.stopPropagation();
      event.preventDefault();
      return true;
   }
   catch(ex)
   {
      console.log("Couldn't upload your pasted image data through clipboardData: " + ex);
      return false;
   }
}

function uploadBlob(blob, bucket, name)
{
  if (!blob) {
    alert("Trying to upload null blob (perhaps browser failed to convert to blob)");
    return;
  } else {
   console.log("Uploading blob: " + blob);
  }
   var data = new FormData();
   data.append("image", blob, name || "blob.png");
   if(bucket) data.append("bucket", bucket);

   var xhr = new XMLHttpRequest();
   xhr.open("POST", SERVERROOT + "/uploadimage", true);
   xhr.onload = function() { location.href = xhr.response; };
   xhr.send(data);
}

//Slideshow stuff
function slideshowElement() { return document.getElementById("slideshow"); }
function slideshowImage() { return document.getElementById("slideshowimage"); }
function slideshowNext() { return document.getElementById("slideshownext"); }
function slideshowBack() { return document.getElementById("slideshowback"); }
function getImage(index) { return document.querySelectorAll(".uploads img")[index]; }

function startSlideshow()
{
   var ss = slideshowElement();
   var ssi = slideshowImage();

   if(!ssi.src)
   {
      slideshowShift(0);
      //ssi.src = getImage(0).src;
   }

   ss.removeAttribute("data-hide");
   launchIntoFullscreen(ss);
}

function endSlideshow()
{
   exitFullscreen();
   slideshowElement().setAttribute("data-hide", "");
}

function slideshowShift(amount)
{
   var ssi = slideshowImage();
   var ssii = Number(ssi.getAttribute("data-imageindex") || 0) + amount;
   ssi.src = getImage(ssii).src;
   ssi.setAttribute("data-imageindex", ssii);
   slideshowNext().disabled = getImage(ssii + 1) ? false : true;
   slideshowBack().disabled = getImage(ssii - 1) ? false : true;
}

//Extras.js
function launchIntoFullscreen(element) 
{
   if(element.requestFullscreen)
      element.requestFullscreen();
   else if(element.mozRequestFullScreen)
      element.mozRequestFullScreen();
   else if(element.webkitRequestFullscreen)
      element.webkitRequestFullscreen();
   else if(element.msRequestFullscreen)
      element.msRequestFullscreen();
}

function exitFullscreen() 
{
   if(document.exitFullscreen)
      document.exitFullscreen();
   else if(document.mozCancelFullScreen)
      document.mozCancelFullScreen();
   else if(document.webkitExitFullscreen)
      document.webkitExitFullscreen();
}

function isFullscreen()
{
   if(document.fullscreenElement || document.mozFullScreenElement ||
      document.webkitFullscreenElement)
      return true;

   return false;
   /*return document.fullscreenEnabled || 
      document.mozFullScreenEnabled || document.webkitFullscreenEnabled;*/
}
