<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
	  <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><?=$app->getTitle()?></h2>
    <p><?=$app->getDescription()?></p>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('app_infopanel',array('act'=>'upload'))?>
  </div>

  <div class="col-xs-12 col-sm-8 col-md-9">
    <form class="form-horizontal" method="post" action="<?=url("/app/upload_post?id={$app->getId()}")?>">
      <div class="form-group">
        <input type="file" class="hidden" id="file-selector">
        <div class="well well-lg droparea text-center hidden-xs">
          Drop your apk/ipa file here.
        </div>
        <label class="control-label col-md-2">File</label>
        <div class="col-md-10">
          <div class="input-group"  id="input-group-icon">
            <input type="text" class="form-control droparea" id="file-name" readonly="readonly">
            <a id="icon-browse" class="input-group-addon btn btn-default">Browse</a>
          </div>
          <div class="help-block">
            <div class="progress progress-striped active">
              <div class="progress-bar" style="width:96%"></div>
            </div>
            <span id="file-info">&nbsp;</span>
          </div>
        </div>
      </div>

      <div class="form-group">
        <label for="title" class="control-label col-md-2">Title</label>
        <div class="col-md-10">
          <input type="text" class="form-control" name="title" id="title">
        </div>
      </div>

      <div class="form-group">
        <label for="description" class="control-label col-md-2">Description</label>
        <div class="col-md-10">
          <textarea class="form-control" row="3" id="description" name="description"></textarea>
        </div>
      </div>

      <div class="form-group">
        <label class="control-label col-md-2">Tags</label>
        <div class="col-md-10">

          <input type="checkbox" class="hidden" name="tags[]" value="ほげ">
          <button class="btn btn-default tags" data-toggle="button">ほげ</button>
          <input type="checkbox" class="hidden" name="tags[]" value="ふが">
          <button class="btn btn-default tags" data-toggle="button">ふが</button>

          <div id="tag-template" class="hidden">
            <input type="checkbox" class="hidden" name="tags[]" value="">
            <button class="btn btn-default tags" data-toggle="button"></button>
          </div>

          <div class="btn-group">
            <a class="btn btn-default dropdown-toggle" href="#" data-toggle="dropdown"><i class="fa fa-plus"></i></a>
            <div id="new-tag-form" class="dropdown-menu">
              <div class="container">
                <input type="text" id="new-tag-name" class="form-control">
                <button id="new-tag-create" class="btn btn-primary">Create</button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="form-group">
        <div class="col-md-10 col-md-offset-2">
          <input type="submit" class="btn btn-primary" value="Upload">
        </div>
      </div>

    </form>
  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

<script type="text/javascript">


// initialize tags button state
$('input[name="tags[]"]').each(function(i,val){
  if($(val).prop('checked')){
    $(val).next().addClass('active');
  }
});
// toggle tags checkbox
$('.btn.tags').on('click',function(event){
  $(this).prev().prop('checked',!$(this).hasClass('active'));
});

// don't close dropdown
$('#new-tag-form').click(function(event){
  event.stopPropagation();
});

// click create button by enter key
$('#new-tag-name').keydown(function(event){
  if(event.keyCode==13){
    $('#new-tag-create').click();
    return false;
  }
  return true;
});

// create new tag button
$('#new-tag-create').on('click',function(event){
  var $tagname = $('#new-tag-name');
  var tag = $tagname.val();
  if(tag){
    var $tmpl = $('#tag-template');
    var $c = $tmpl.children().clone(true);

    $($c[0]).attr('value',tag).prop('checked',true);
    $($c[1]).text(tag).addClass('active')

    $tmpl.before($c);
    $tmpl.before(' ');

    $tagname.val(null);
  }
  $('.dropdown-toggle').parent().removeClass('open');
  return false;
});


</script>