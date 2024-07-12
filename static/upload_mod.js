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

function removeUpload() {
  $('.file-upload-input').replaceWith($('.file-upload-input').clone());
  $('.file-upload-content').hide();
  $('.image-upload-wrap').show();
}
$('.image-upload-wrap').bind('dragover', function () {
  $('.image-upload-wrap').addClass('image-dropping');
});
$('.image-upload-wrap').bind('dragleave', function () {
  $('.image-upload-wrap').removeClass('image-dropping');
});
