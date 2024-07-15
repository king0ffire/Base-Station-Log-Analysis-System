function readURL(input) {
  if (input.files && input.files[0]) {

    var ajax = new XMLHttpRequest();
    var formdata = new FormData();
    formdata.append("uploadfile", input.files[0]);
    ajax.open("POST", "/upload");
    ajax.onload = function () {
      if (ajax.status === 200) {
        console.log('上传成功');
      } else {
        console.log('出错了');
      }
    };
    $('#uploadprogress').show();
    ajax.upload.onprogress = function (event) {
      if (event.lengthComputable) {
        var complete = (event.loaded / event.total * 100 | 0);
        var progress = document.getElementById('uploadprogress');
        progress.value = progress.innerHTML = complete;
      }
    };
  ajax.onreadystatechange = function () {
    if(ajax.readyState == 4)
    {
      window.location.href = ajax.responseURL
    }
}
  };
  ajax.send(formdata);
  
}

