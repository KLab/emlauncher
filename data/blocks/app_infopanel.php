<div class="list-group">
<?php if($app->isOwner($login_user)): ?>
  <div class="list-group-item">
    <ul class="nav nav-pills nav-stacked">
      <li<?=($action==='upload')?' class="active"':''?>>
        <a href="<?=url("/app/upload?id={$app->getId()}")?>"><i class="fa fa-upload"></i> Upload</a>
      </li>
      <li<?=($action==='preferences')?' class="active"':''?>>
        <a href="<?=url("/app/preferences?id={$app->getId()}")?>"><i class="fa fa-wrench"></i> Preferences</a>
      </li>
    </ul>
  </div>
<?php endif ?>
  <div class="list-group-item">
    <dl>
<?php if($app->getLastUpload()): ?>
      <dt>last upload</dt>
      <dd><?=$app->getLastUpload()?></dd>
<?php endif ?>
      <dt>created</dt>
      <dd><?=$app->getCreated()?></dd>
      <dt>install user</dt>
<?php if($app->isOwner($login_user)): ?>
      <dd>
        <div class="dropdown">
          <a class="dropdown-toggle" id="install-user-count" data-toggle="dropdown">
            <?=$app->getInstallUserCount()?>
          </a>
          <ul class="dropdown-menu" role="menu" aria-labelledby="install-user-count">
<?php foreach($app->getInstallUsers() as $u): ?>
            <li role="presentation"><a role="menuitem" tabindex="-1"><?=$u->getMail()?></a></li>
<?php endforeach ?>
          </ul>
        </div>
      </dd>
<?php else: ?>
      <dd><?=$app->getInstallUserCount()?></dd>
<?php endif ?>
<?php if($app->getRepository()): ?>
      <dt>repository</dt>
<?php if(preg_match('|https?://([^/]*)/|',$app->getRepository(),$m)):?>
      <dd>
        <a target="_blank" href="<?=htmlspecialchars($app->getRepository())?>">
<?php if(strpos($m[1],'github')!==false): ?>
          <i class="fa fa-github"></i>
<?php elseif(strpos($m[1],'bitbucket')!==false): ?>
          <i class="fa fa-bitbucket"></i>
<?php endif ?>
          <?=htmlspecialchars($m[1])?>
        </a>
      </dd>
<?php else: ?>
      <dd><input type="text" class="form-control" readonly="readonly" value="<?=htmlspecialchars($app->getRepository())?>"></dd>
<?php endif ?>
<?php endif ?>
      <dt>owners</dt>
<?php foreach($app->getOwners() as $owner):?>
      <dd><a href="mailto:<?=$owner->getOwnerMail()?>"><?=$owner->getOwnerMail()?></a></dd>
<?php endforeach ?>
    </dl>
  </div>
  <div class="list-group-item">
    <div class="text-center">
      <p>link to this app</p>
      <img src="http://chart.apis.google.com/chart?chs=150&cht=qr&chl=<?=urlencode(url("/app?id={$app->getId()}"))?>">
    </div>
  </div>
</div>
