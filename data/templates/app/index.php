
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

    <table id="package-list" class="table table-striped">
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
          <a href="<?=url('/package?id='.$pkg->getId())?>"><?=$pkg->getTitle()?></a><br>
          <?=$pkg->getCreated('Y-m-d H:i')?>
        </td>
        <td>
<?php foreach($pkg->getTags() as $tag): ?>
          <span class="label label-info"><?=$tag->getName()?></span>
<?php endforeach ?>
        </td>
        <td class="text-center">
          <a class="btn btn-primary install-link" href="<?=url('/package/install?id='.$pkg->getId())?>">Install</a>
        </td>
      </tr>
<?php endforeach ?>
    </table>

  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

