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
<?endif?>
  <div class="list-group-item">
    <dl>
<?php if($app->getLastUpload()): ?>
      <dt>last upload</dt>
      <dd><?=$app->getLastUpload()?></dd>
<?php endif ?>
      <dt>created</dt>
      <dd><?=$app->getCreated()?></dd>
<?php if($app->getRepository()): ?>
      <dt>repository</dt>
<?php if(strpos($app->getRepository(),'http')===0): ?>
      <dd><a href="<?=htmlspecialchars($app->getRepository())?>"><?=htmlspecialchars($app->getRepository())?></a></dd>
<?php else: ?>
      <dd><?=htmlspecialchars($app->getRepository())?></dd>
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
