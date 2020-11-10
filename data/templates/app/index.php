
<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a></h2>
    <p><?=nl2br(htmlspecialchars($app->getDescription()))?></p>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('app_infopanel')?>
  </div>


  <div class="col-xs-12 col-sm-8 col-md-9">

    <div id="comments">
      <div class="row">
        <div class="col-xs-6">
          <h3><?=$comment_count?> comments</h3>
        </div>
        <div class="col-xs-6 text-right">
          <a href="<?=url("/app/comment?id={$app->getId()}")?>" class="btn btn-sm btn-default"><i class="fa fa-pencil"></i> write a comment</a>
        </div>
      </div>
<?php if($comment_count>0): ?>
      <ul class="list-group">
<?php
foreach($top_comments as $c):
    $pkg = ($c->getPackageId())? $commented_package[$c->getPackageId()]: null;
?>
        <li class="list-group-item">
          <dl>
            <dt><a href="<?=url("/app/comment?id={$app->getId()}#comment-{$c->getNumber()}")?>"><?=$c->getNumber()?></a></dt>
            <dd><?=htmlspecialchars($c->getMessage())?></dd>
          </dl>
          <div class="text-right">
<?php if($pkg): ?>
            <a href="<?=url("/package?id={$pkg->getId()}")?>">
              <?=block('platform_icon',array('package'=>$pkg))?> <?=htmlspecialchars($pkg->getTitle())?></a>
<?php else: ?>
            <span>No package installed</span>
<?php endif ?>
            (<?=$c->getCreated('Y-m-d H:i')?>)
          </div>
        </li>
<?php endforeach ?>
      </ul>
      <div class="text-right">
        <a href="<?=url("/app/comment?id={$app->getId()}#comments")?>">read more...</a>
      </div>
<?php endif ?>
    </div>

    <ul id="pf-nav-tabs" class="nav nav-tabs">
      <li<?php if($pf==='android'):?> class="active"<?php endif?> id="android">
        <a href="<?="?id={$app->getId()}&pf=android"?>">Android</a>
      </li>
      <li<?php if($pf==='ios'):?> class="active"<?php endif?> id="ios">
        <a href="<?="?id={$app->getId()}&pf=ios"?>">iOS</a>
      </li>
      <li<?php if($pf==='other'):?> class="active"<?php endif?> id="other">
        <a href="<?="?id={$app->getId()}&pf=other"?>">Other</a>
      </li>
      <li<?php if($pf==='all'):?> class="active"<?php endif?> id="all">
        <a href="<?="?id={$app->getId()}&pf=all"?>">All</a>
      </li>
    </ul>

    <div id="tag-filter">
      <a id="tag-filter-toggle" class="pull-right badge"><i class="fa fa-angle-double-<?=$filter_open?'up':'down'?>"></i></a>
      <div id="tag-filter-body" style="display: <?=($filter_open)? 'block': 'none'?>">
<?php foreach($app->getTags() as $tag): ?>
        <button id="<?=$tag->getId()?>" class="btn btn-default <?=in_array($tag->getId(), $active_tags) ? 'on active' : '' ?>" data-toggle="button">
        <?=htmlspecialchars($tag->getName())?></button>
<?php endforeach ?>
      </div>
    </div>

    <table id="package-list" class="table table-hover">
<?php foreach($packages as $pkg): ?>
      <tr>
        <td class="text-center logo">
          <?=block('platform_icon',array('package'=>$pkg))?>
        </td>
        <td class="package-list-item-info">
          <div class="row">
            <div class="col-xs-12 col-md-7">
<?php if($pkg->isProtected()): ?>
              <i class="fa fa-lock"></i>
<?php endif ?>
              <a class="title" href="<?=url('/package?id='.$pkg->getId())?>"><?=htmlspecialchars($pkg->getTitle())?></a>
<?php if($pkg->getDescription()):?>
              <p class="text-muted description"><?=$pkg->getShortDescription()?></p>
<?php endif?>
<?php
    $units = array('B','KB','MB','GB');
    $size = $pkg->getFileSize();
    for($i=0;$i<count($units);$i++){
        if($size<1024) break;
        $size = round($size/1024, 1);
    }
?>
<?php if($pkg->getIdentifier()): ?>
              <span class="info hidden-xs hidden-sm"><?=$pkg->getIdentifier()?></span>
<?php endif?>
              <span class="info hidden-xs hidden-sm"><?=$pkg->getFileSize()?"{$size} {$units[$i]}":'--'?>, <?=$pkg->getCreated('Y-m-d H:i')?></span>
            </div>
            <div class="col-xs-12 col-md-5">
<?php if($pkg->isFileSizeWarned()): ?>
              <span class="label label-danger">Over <?=$pkg->getFileSizeLimitMB()?> MB</span>
<?php endif ?>
<?php foreach($pkg->getTags() as $tag): ?>
              <span class="label label-default" data="<?=htmlspecialchars($tag->getName())?>"><?=htmlspecialchars($tag->getName())?></span>
<?php endforeach ?>
            </div>
          </div>
<?php if($pkg->getIdentifier()): ?>
          <span class="info visible-xs visible-sm"><?=$pkg->getIdentifier()?></span>
<?php endif?>
          <span class="info visible-xs visible-sm"><?=$pkg->getFileSize()?"{$size} {$units[$i]}":'--'?></span>
          <span class="info visible-xs visible-sm"><?=$pkg->getCreated('Y-m-d H:i')?></span>
        </td>
        <td class="text-center">
<?php if($login_user->getPackageInstalledDate($pkg)): ?>
          <a class="btn btn-success install-link col-xs-12" href="<?=$pkg->getInstallUrl()?>"><i class="fa fa-check"></i> Installed</a>
<?php else: ?>
          <a class="btn btn-primary install-link col-xs-12" href="<?=$pkg->getInstallUrl()?>"><i class="fa fa-download"></i> Install</a>
<?php endif ?>
        </td>
      </tr>
<?php endforeach ?>
    </table>

    <ul class="pager">
<?php if($current_page==1): ?>
      <li class="previous disabled"><span>Previous</span></li>
<?php else: ?>
      <li class="previous"><a href="<?=mfwHttp::composeURL(mfwRequest::url(),array('page'=>$current_page-1))?>">Previous</a></li>
<?php endif ?>

<?php if($has_next_page):?>
      <li class="next"><a href="<?=mfwHttp::composeURL(mfwRequest::url(),array('page'=>$current_page+1))?>">Next</a></li>
<?php else: ?>
      <li class="next disabled"><span>Next</span></li>
<?php endif ?>
    </ul>

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
  $('.pager>li>a').each(function(){
    if($up.length>0){
      this.href = this.href.replace(/&filter_open=1/,'');
    }
    else{
      this.href = this.href + '&filter_open=1';
    }
  });

  $('#tag-filter-body').slideToggle('fast');
});

function get_url_param_tabs() {
  var $active_tags = $('#tag-filter-body>button.on');
  if ($active_tags.length>0) {
    var tags = '';
    $active_tags.each(function(i){tags += $active_tags[i].id + '+';});
    return '&tags=' + tags.substring(0, tags.length - 1);
  } else {
    return '';
  }
}

function compose_url() {
  var pf = 'all';
  var $active_pf_tabs = $('#pf-nav-tabs>li.active');
  if ($active_pf_tabs.length>0) {
    pf = $active_pf_tabs[0].id
  }
  var of = '';
  if ($('i.fa-angle-double-up').length>0) {
    of = '&filter_open=1';
  }
  return "<?="id={$app->getId()}&pf="?>" + pf + get_url_param_tabs() + of;
}

// filter by tag
$('#tag-filter-body>button').on('click',function(){
  if($(this).hasClass('on')){
    $(this).removeClass('on');
  }
  else{
    $(this).addClass('on');
  }
  location.href = '?' + compose_url();
});

$('.package-list-item-info').on('click',function(event){
  $('a',this)[0].click();
});

$('#pf-nav-tabs>li').on('click', function(event){
  if ($('a', this)) {
    location.href = $('a', this)[0].href + get_url_param_tabs();
    event.preventDefault();
  }
});

</script>
