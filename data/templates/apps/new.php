
<div class="well">
  <form class="form-horizontal" method="post" action="<?=url('/apps/create')?>" enctype="multipart/form-data">
    <fieldset>
      <legend>New application</legend>

      <div class="row">
        <div class="col-md-10 col-sm-9">
          <div class="form-group">
            <label for="title" class="control-label col-md-2">Title</label>
            <div class="col-md-10">
              <div id="alert-notitle" class="alert alert-danger hidden">
                タイトルが入力されていません
              </div>
              <input class="form-control" type="text" id="title" name="title">
            </div>
          </div>
        
          <div class="form-group">
            <label for="icon-text" class="control-label col-md-2">Icon</label>
            <div class="col-md-10">
              <div id="alert-noicon" class="alert alert-danger hidden">
                アイコン画像が指定されていません
              </div>
              <input type="file" id="icon" class="hidden">
              <div class="input-group">
                <input type="text" class="form-control" id="icon-text" disabled="disabled">
                <a id="icon-browse" class="input-group-addon btn btn-default">Browse</a>
              </div>
            </div>
          </div>
        </div>
      
        <div class="col-md-2 col-sm-3 hidden-xs">
          <img id="icon-preview" class="img-thumbnail" style="width:96px;height:96px;">
        </div>
      </div>
      
      <div class="row">
        <div class="form-group">
          <div class="col-md-10">
            <label for="description" class="control-label col-md-2">Description</label>
            <div class="col-md-10">
              <textarea class="form-control" row="3" id="description" name="description"></textarea>
            </div>
          </div>
        </div>
      </div>

      <div class="row">
        <div class="form-group">
          <div class="col-md-10">
            <div class="col-md-10 col-md-offset-2">
              <input type="submit" class="btn btn-primary" value="Create">
            </div>
          </div>
        </div>
      </div>

    </fieldset>
  </form>

</div>

<script type="text/javascript">

$('#icon-browse').on('click',function(event){
  $('#icon').click();
  return false;
});

$('#icon-preview').on('click',function(event){
  $('#icon').click();
  return false;
});

$('#icon').on('change',function(event){
  var file = event.target.files[0];
  if(!file || !file.type.match('image.*')){
    $(this).val(null);
    $('#icon-text').val(null);
    $('#icon-preview').attr('src','data:image/gif;base64,R0lGODlhAQABAIABAP///wAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw==');//透明gif
    return false;
  }
  var reader = new FileReader;
  reader.onload = function(e){
    $('#icon-preview').attr('src',e.target.result);
  };
  reader.readAsDataURL(file);
  $('#icon-text').val(file.name);
});

$('form').submit(function(){
  var valid = true;
  $('.alert').addClass('hidden');
  if($('#title').val()==''){
    $('#alert-notitle').removeClass('hidden');
    valid = false;
  }
  if(!$('#icon').val()){
    $('#alert-noicon').removeClass('hidden');
    valid = false;
  }
  return valid;
});


</script>