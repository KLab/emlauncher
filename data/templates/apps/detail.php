
<div class="media">
  <p class="pull-left">
    <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
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

    <ul>
      <li><i class="fa fa-android"></i>binary informations</li>
      <li><i class="fa fa-android"></i>binary informations</li>
      <li><i class="fa fa-apple"></i>binary informations</li>
      <li><i class="fa fa-android"></i>binary informations</li>
      <li><i class="fa fa-apple"></i>binary informations</li>
    </ul>
  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

