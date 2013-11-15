
<div class="well">
  <form class="form-horizontal" method="post" action="<?=url('/app/create')?>" enctype="multipart/form-data">
    <fieldset>
      <legend>New application</legend>

      <div class="row">
        <div class="col-md-10 col-sm-9">
          <div class="form-group">
	        <label for="title" class="control-label col-md-2 required">Title</label>
            <div class="col-md-10">
              <div id="alert-notitle" class="alert alert-danger hidden">
                タイトルが入力されていません
              </div>
              <input class="form-control" type="text" id="title" name="title">
            </div>
          </div>

          <div class="form-group">
            <label for="icon-selector" class="control-label col-md-2 required">Icon</label>
            <div class="col-md-10">
              <div id="alert-icon-size-limit" class="alert alert-danger hidden">
                画像ファイルサイズが大きすぎます
              </div>
              <div id="alert-noicon" class="alert alert-danger hidden">
                アイコン画像が指定されていません
              </div>
              <input type="hidden" id="icon-data" name="icon-data" value="">
              <input type="file" id="icon-selector" class="hidden">
              <div class="input-group"  id="input-group-icon">
                <input type="text" class="form-control droparea" id="icon-text" readonly="readonly">
                <a id="icon-browse" class="input-group-addon btn btn-default">Browse</a>
              </div>
            </div>
          </div>
        </div>

        <div class="col-md-2 col-sm-3 hidden-xs text-center">
          <img id="icon-preview" class="img-thumbnail droparea" style="width:96px;height:96px;">
        </div>
      </div>

      <div class="row">
        <div class="col-md-10">
          <div class="form-group">
            <label for="description" class="control-label col-md-2">Description</label>
            <div class="col-md-10">
              <textarea class="form-control" row="3" id="description" name="description"></textarea>
            </div>
          </div>
        </div>
      </div>

      <div class="row">
        <div class="col-md-10">
          <div class="form-group">
            <label for="repository" class="control-label col-md-2">Repository</label>
            <div class="col-md-10">
              <input type="text" class="form-control" id="repository" name="repository">
            </div>
          </div>
        </div>
      </div>

      <div class="row">
        <div class="col-md-10">
          <div class="form-group">
            <div class="col-md-10 col-md-offset-2">
              <button class="btn btn-primary"><i class="fa fa-save"></i> Create</button>
            </div>
          </div>
        </div>
      </div>

    </fieldset>
  </form>

</div>

<script type="text/javascript">
$(document).on('drop dragover',function(e){e.preventDefault()});

$('#icon-browse').on('click',function(event){
  $('#icon-selector').click();
  return false;
});
$('#icon-preview').on('click',function(event){
  $('#icon-selector').click();
  return false;
});

function setIconFile(file){
  $('#icon-data').val(null);
  $('#icon-text').val(null);
  $('#icon-preview').attr('src','data:image/gif;base64,R0lGODlhAQABAIABAP///wAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==');//透明gif
  $('#alert-icon-size-limit').addClass('hidden');

  if(!file || !file.type.match('^image/(png|gif|jpeg)$')){
    return false;
  }
  if(file.size > 1000000){
    $('#alert-icon-size-limit').removeClass('hidden');
    return false;
  }

  var reader = new FileReader;
  reader.onload = function(e){
    $('#icon-data').val(e.target.result);
    $('#icon-preview').attr('src',e.target.result);
  };
  reader.readAsDataURL(file);
  $('#icon-text').val(file.name);
}

$('#icon-preview').on('drop',function(event){
  var file = event.originalEvent.dataTransfer.files[0];
  $('.droparea').removeClass('dragover');
  return setIconFile(file);
});
$('#input-group-icon').on('drop',function(event){
  var file = event.originalEvent.dataTransfer.files[0];
  $('.droparea').removeClass('dragover');
  return setIconFile(file);
});
$('.droparea').on('dragenter',function(event){
  $('.droparea').removeClass('dragover');
  $(this).addClass('dragover');
});
$('.droparea').on('dragleave',function(event){
  $(this).removeClass('dragover');
});

$('#icon-selector').on('change',function(event){
  var file = event.target.files[0];
  return setIconFile(file);
});

$('form').submit(function(){
  var valid = true;
  $('.alert').addClass('hidden');
  if($('#title').val()==''){
    $('#alert-notitle').removeClass('hidden');
    valid = false;
  }
  if(!$('#icon-text').val()){
    $('#alert-noicon').removeClass('hidden');
    valid = false;
  }
  return valid;
});


</script>
