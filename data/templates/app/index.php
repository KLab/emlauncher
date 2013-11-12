
<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><?=htmlspecialchars($app->getTitle())?></h2>
    <p><?=htmlspecialchars($app->getDescription())?></p>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('app_infopanel')?>
  </div>


  <div class="col-xs-12 col-sm-8 col-md-9">
    <ul class="nav nav-tabs">
      <li<?php if($pf==='android'):?> class="active"<?php endif?>>
        <a href="<?="?id={$app->getId()}&pf=android"?>">Android</a>
      </li>
      <li<?php if($pf==='ios'):?> class="active"<?php endif?>>
        <a href="<?="?id={$app->getId()}&pf=ios"?>">iOS</a>
      </li>
      <li<?php if($pf==='all'):?> class="active"<?php endif?>>
        <a href="<?="?id={$app->getId()}&pf=all"?>">All</a>
      </li>
    </ul>

    <div id="tag-filter">
      <a id="tag-filter-toggle" class="pull-right badge"><i class="fa fa-angle-double-down"></i></a>
      <div id="tag-filter-body" style="display:none">
<?php foreach($app->getTags() as $tag): ?>
	    <button class="btn btn-default" data-toggle="button"><?=htmlspecialchars($tag->getName())?></button>
<?php endforeach ?>
      </div>
    </div>

    <table id="package-list" class="table table-hover">
<?php foreach($packages as $pkg): ?>
      <tr>
        <td class="text-center logo">
<?php if($pkg->getPlatform()==='Android'): ?>
          <i class="fa fa-android"></i>
<?php elseif($pkg->getPlatform()==='iOS'): ?>
          <i class="fa fa-apple"></i>
<?php else: ?>
          <i class="fa fa-question"></i>
<?php endif ?>
        </td>
        <td>
	      <div class="row">
            <div class="col-xs-12 col-md-6">
              <a class="title" href="<?=url('/package?id='.$pkg->getId())?>"><?=htmlspecialchars($pkg->getTitle())?></a>
              <span class="info hidden-xs hidden-sm"><?=$pkg->getCreated('Y-m-d H:i')?></span>
            </div>
            <div class="col-xs-12 col-md-6">
<?php foreach($pkg->getTags() as $tag): ?>
              <span class="label label-default" data="<?=htmlspecialchars($tag->getName())?>"><?=htmlspecialchars($tag->getName())?></span>
<?php endforeach ?>
            </div>
          </div>
          <span class="info visible-xs visible-sm"><?=$pkg->getCreated('Y-m-d H:i')?></span>
        </td>
        <td class="text-center">
          <a class="btn btn-primary install-link col-xs-12" href="<?=$pkg->getInstallUrl()?>"><i class="fa fa-download"></i> Install</a>
        </td>
      </tr>
<?php endforeach ?>
    </table>

  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

<script type="text/javascript">

$('#tag-filter-toggle').on('click',function(){
  $down = $('i.fa-angle-double-down');
  $up = $('i.fa-angle-double-up');
  if($down){
    $down.removeClass('fa-angle-double-down').addClass('fa-angle-double-up');
  }
  if($up){
    $up.removeClass('fa-angle-double-up').addClass('fa-angle-double-down');
  }
  $('#tag-filter-body').slideToggle('fast');
});

// filter by tag
$('#tag-filter-body>button').on('click',function(){
  if($(this).hasClass('on')){
    $(this).removeClass('on');
  }
  else{
    $(this).addClass('on');
  }
  var $active_tags = $('#tag-filter-body>button.on');

  $list = $('#package-list tr').removeClass('hidden');

  if($active_tags.length>0){
    $active_tags.each(function(){
      $list.not(':has(.label[data="'+$(this).text()+'"])').addClass('hidden');
      $list = $list.not('.hidden');
    });
  }

});



</script>
