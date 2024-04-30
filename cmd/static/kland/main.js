(function(){

window.addEventListener("load", onLoad);

var rightArrow = ">|\&gt;";

function now()
{
   if(Performance && Performance.now)
      return Performance.now();
   else
      return Date.now();
}

//Loading will perform one-time setup for whatever.
function onLoad(event)
{
   var time = now();
   var i, j;

   //Replace spoilers 
   var spoilers = document.querySelectorAll("sp");

   for(i = 0; i < spoilers.length; i++)
   {
      var newSpoiler = document.createElement("s-p");
      newSpoiler.innerHTML = spoilers[i].innerHTML;
      spoilers[i].parentNode.insertBefore(newSpoiler, spoilers[i]);
      spoilers[i].parentNode.removeChild(spoilers[i]);
   }

   //Fix up time elements so they don't show some sucky time.
   var times = document.querySelectorAll("time");

   for(i = 0; i < times.length; i++)
   {
      var dateTime = new Date(times[i].getAttribute("datetime"));
      times[i].innerHTML = dateTime.toLocaleString();
   }

   //Fix up images so they're links to the originals.
   var images = document.querySelectorAll(".post img");

   for(i = 0; i < images.length; i++)
   {
      var link = document.createElement("a");
      link.href = images[i].src;
      images[i].parentNode.insertBefore(link, images[i]);
      link.appendChild(images[i]);
   }

   //Gather the available post IDs and links
   var postLinks = document.querySelectorAll(".post .postlink");
   var idRegex = new RegExp("([0-9]+)");
   var postIDs = {};

   for(i = 0; i < postLinks.length; i++)
   {
      var postID = idRegex.exec(postLinks[i].innerHTML)[1];
      var referenceLinks = document.createElement("div");
      referenceLinks.className = "references";
      insertAfter(postLinks[i].parentNode, referenceLinks, postLinks[i]);

      postIDs[postID] = 
      {
         "id" : postID,
         "link" : postLinks[i].href,
         "references" : referenceLinks,
         "op" : (i === 0)
      };
   }

   //Fix up greentext and linkbacks
   var posts = document.querySelectorAll(".post .content");
   var gtRegex = new RegExp("^\s*(" + rightArrow + ")");
   var lnRegex = new RegExp("^\s*(" + rightArrow + ")(" + rightArrow + ")([0-9]+)");
   var parser = new Org.Parser();
   var parsedOrg = false;

   for(i = 0; i < posts.length; i++)
   {
      //{translateSymbolArrow:true,supressCheckboxHandling:true}).contentHTML;

      var lines = posts[i].innerHTML.split("\n");
      posts[i].innerHTML = "";

      for(j = 0; j < lines.length; j++)
      {
         var line = lines[j];
         var lnMatch = lnRegex.exec(line);

         //We want to try to match post references first, since they're
         //included in the "greentext" set and thus would get gobbled up.
         if(lnMatch && lnMatch[3] && postIDs[lnMatch[3]])
         {
            line = createLinkFromPostData(postIDs[lnMatch[3]], line);

            //Only add referenced posts if our post has a data item 
            if(posts[i].dataset && posts[i].dataset.pid)
            {
               postIDs[lnMatch[3]].references.innerHTML +=
                  createLinkFromPostData(postIDs[posts[i].dataset.pid], ">>" + posts[i].dataset.pid)+ " ";
            }
         }
         else if(gtRegex.test(line))
         {
            line = '<span class="greentext">' + line + '</span>';
         }

         posts[i].innerHTML += (j > 0 ? "\n" : "") + line;
      }

      parsedOrg = parser.parse("fake line\n\n" + posts[i].innerHTML);
      posts[i].innerHTML = HTMLUtilities.UnescapeHTML(parsedOrg.convert(Org.ConverterHTML,{suppressAutoLink:true}).contentHTML);
      //posts[i].innerHTML = parsedOrg.convert(Org.ConverterHTML,{}).contentHTML;
   }

   time = now() - time;
   console.log("OnLoad took: " + time + "ms");
}

function createLinkFromPostData(postData, text)
{
   return '<a class="reference" href="' + postData.link + '">' + 
      text + (postData.op ? " (OP)" : "") + '</a>';
}

function insertAfter(parentNode, newNode, afterElement) 
{
   if(parentNode.lastChild === afterElement)
      parentNode.appendChild(newNode);
   else
      parentNode.insertBefore(newNode, afterElement.nextSibling);
}

})();
