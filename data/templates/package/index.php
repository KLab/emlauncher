<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a></h2>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('pkg_infopanel')?>
  </div>

  <div class="col-xs-12 col-sm-8 col-md-9">

    <div class="row">
      <div class="col-xs-7">
        <h3>
          <a href="<?=url("/package?id={$package->getId()}")?>">
            <?=block('platform_icon')?>
            <?=htmlspecialchars($package->getTitle())?>
          </a>
<?php if($package->isProtected()): ?>
          <i class="fa fa-lock"></i>
<?php endif ?>
        </h3>
        <div id="description">
          <div class="read-more">
            <a class="badge">more...</a>
          </div>
          <p>
            <?=nl2br(htmlspecialchars($package->getDescription()))?>
          </p>
        </div>
      </div>
      <div class="col-xs-5">
<?php if($login_user->getPackageInstalledDate($package)): ?>
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-success col-xs-12"><i class="fa fa-check"></i> Installed</a>
        <dl id="installed-date">
          <dt>Instaled at</dt>
          <dd><?=$login_user->getPackageInstalledDate($package)?></dd>
        </dl>
<?php else: ?>
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-primary col-xs-12"><i class="fa fa-download"></i> Install</a>
<?php endif ?>
      </div>
    </div>

    <p>
<?php if($package->isFileSizeWarned()): ?>
      <span class="label label-danger">Over <?=$package->getFileSizeLimitMB()?> MB</span>
<?php endif ?>
<?php foreach($package->getTags() as $tag): ?>
      <span class="label label-default"><?=htmlspecialchars($tag->getName())?></span>
<?php endforeach ?>
    </p>

    <dl class="dl-horizontal">
      <dt>Platform</dt>
      <dd><?=block('platform_icon',array('with_name'=>true))?></dd>
<?php if($package->getIdentifier()): ?>
      <dt>Bundle ID</dt>
      <dd><?=$package->getIdentifier()?></dd>
<?php endif ?>
      <dt>Original name</dt>
      <dd><?=$package->getOriginalFileName()?:'--------.'.pathinfo($package->getBaseFileName(),PATHINFO_EXTENSION)?></dd>
      <dt>File size</dt>
<?php
    $units = array('B','KB','MB','GB');
    $size = $package->getFileSize();
    for($i=0;$i<count($units);$i++){
        if($size<1024) break;
        $size = round($size/1024, 1);
    }
?>
      <dd><?=$size?"{$size} {$units[$i]}":'-'?></dd>
      <dt>Install user</dt>
<?php if($app->isOwner($login_user)): ?>
      <dd>
        <div class="dropdown">
          <a class="dropdown-toggle" id="install-user-count" data-toggle="dropdown">
            <?=$package->getInstallCount()?>
          </a>
          <ul class="dropdown-menu" role="menu" aria-labelledby="install-user-count">
<?php foreach($package->getInstallUsers() as $mail): ?>
            <li role="presentation"><a role="menuitem" tabindex="-1"><?=$mail?></a></li>
<?php endforeach ?>
          </ul>
        </div>
      </dd>
<?php else: ?>
      <dd><?=$package->getInstallCount()?></dd>
<?php endif ?>
      <dt>Uploaded</dt>
      <dd><?=$package->getCreated()?></dd>
      <dt>Owners</dt>
<?php foreach($app->getOwners() as $owner):?>
        <dd><a href="mailto:<?=$owner->getOwnerMail()?>"><?=$owner->getOwnerMail()?></a></dd>
<?php endforeach ?>
    </dl>

    <div class="col-xs-12 col-sm-9">
      <p class="text-center">
        <a class="btn btn-default" href="<?=url("/package/create_token?id={$package->getId()}")?>"><i class="fa fa-bolt"></i> Create Install Token</a>&nbsp;
<?php if ($app->isOwner($login_user)):?>
        <a class="btn btn-default" href="<?=url("/package/create_guestpass?id={$package->getId()}")?>"><i class="fa fa-users"></i> Create GuestPass</a>
<?php endif; ?>
      </p>
    </div>

<?php if ($app->isOwner($login_user)):?>
      <div class="col-xs-12 col-sm-9">
          <h3>GuestPass</h3>
          <table class="table  table-striped table-bordered">
            <thead>
                <tr>
                    <th>Expire date</th>
                    <th>Install count</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
<?php foreach($package->getGuestPasses() as $guest_pass): /* @var GuestPass $guest_pass */ ?>
                <tr>
                    <td><a href="<?=url("/package/guestpass?id={$guest_pass->getPackageId()}&guestpass_id={$guest_pass->getId()}")?>"><?=$guest_pass->getExpired()?></td>
                    <td><?=$guest_pass->getInstallCount()?> installed</td>
                    <td><a class="btn btn-default btn-xs" href="<?=url("/package/expire_guestpass?id={$guest_pass->getPackageId()}&guestpass_id={$guest_pass->getId()}")?>">Expire</a></td>
                </tr>
<?php endforeach; ?>
            </tbody>
          </table>
      </div>
<?php endif;// end of guest pass ?>

    <div class="col-xs-12 col-sm-9">
<?php if($package->getAttachedFiles()->count()>0):?>
      <h3>Attached Files</h3>
      <table id="attached-files" class="table table-hover">
<?php foreach($package->getAttachedFiles() as $afile):
    $units = array('B','KB','MB','GB');
    $size = $afile->getFileSize();
    for($i=0;$i<count($units);$i++){
        if($size<1024) break;
        $size = round($size/1024, 1);
    }
?>
        <tr>
          <td>
            <div class="col-xs-12 col-md-8"><?=htmlspecialchars($afile->getOriginalFileName())?></div>
            <div class="col-xs-12 col-md-4 text-center"><?=$size?> <?=$units[$i]?></div>
          </td>
          <td class="text-center">
            <a class="btn btn-default" href="<?=url("/package/attach_download?id={$package->getId()}&attached_id={$afile->getId()}")?>">
              <span class="hidden-xs"><i class="fa fa-download"></i> Download</span>
              <span class="visible-xs"><i class="fa fa-download"></i> DL</span>
            </a>
<?php if($app->isOwner($login_user)):?>
            <a class="btn btn-danger" href="<?=url("/package/attach_delete_confirm?id={$package->getId()}&attached_id={$afile->getId()}")?>"><i class="fa fa-trash-o"></i></a>
<?php endif;?>
          </td>
        </tr>
<?php endforeach;?>
      </table>
<?php endif;?>
<?php if($app->isOwner($login_user)):?>
      <p>
        <button id="attach-button" class="btn btn-default">
          <i id="attach-icon" class="fa fa-upload"></i> Attache a File
        </button>
      </p>
      <form id="attach-form" enctype="multipart/form-data" method="post" action="<?=url("/package/attach")?>">
        <input type="hidden" name="id" value="<?=$package->getId()?>">
        <input type="file" class="hidden" id="attach-file" name="file">
      </form>
<?php endif;?>
    </div>

  </div>
</div>

<div class="visible-xs">
  <?=block('pkg_infopanel')?>
</div>

<script type="text/javascript">
$('#attach-button').on('click',function(event){
  $('#attach-file').click();
  return false;
});

$('#attach-file').on('change',function(event){
  $('#attach-icon').removeClass('fa-upload').addClass('fa-spinner fa-pulse');
  $('#attach-button').attr('disabled', true);
  $('#attach-form').submit();
  return false;
});

if($('#description').height()>200){
  $('#description>.read-more').addClass('active');
  $('#description>.read-more>a').on('click',function(event){
    $('#description>.read-more').removeClass('active');
  });
}
</script>
