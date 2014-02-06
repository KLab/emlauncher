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
        </h3>
        <p>
          <?=nl2br(htmlspecialchars($package->getDescription()))?>
        </p>
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
      <dt>Original name</dt>
      <dd><?=$package->getOriginalFileName()?:'--------.'.pathinfo($package->getBaseFileName(),PATHINFO_EXTENSION)?></dd>
      <dt>File size</dt>
      <dd><?=$package->getFileSize()?number_format($package->getFileSize()):'-'?> bytes</dd>
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
        <a class="btn btn-default" href="<?=url("/package/create_token?id={$package->getId()}")?>"><i class="fa fa-bolt"></i> Create Install Token</a>
      </p>
    </div>

  </div>
</div>

<div class="visible-xs">
  <?=block('pkg_infopanel')?>
</div>

