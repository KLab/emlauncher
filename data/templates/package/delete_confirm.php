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

    <h3>
      <a href="<?=url("/package?id={$package->getId()}")?>">
        <?=block('platform_icon')?>
        <?=htmlspecialchars($package->getTitle())?>
      </a>
<?php if($package->isProtected()): ?>
      <i class="fa fa-lock"></i>
<?php endif ?>
    </h3>

    <div class="col-md-8 col-md-offset-1">
      <div class="panel panel-danger">
        <div class="panel-heading">
          <h2 class="panel-title">Delete Package</h2>
        </div>
        <div class="panel-body">
          <p>インストールパッケージを削除します。この操作は取り消せません。</p>
          <p>本当に削除しますか？</p>
          <div class="text-center">
            <form method="post" action="<?=url('/package/delete')?>">
              <input type="hidden" name="id" value="<?=$package->getId()?>">
              <input type="hidden" name="token" value="<?=$token?>">
              <button class="btn btn-danger"><i class="fa fa-trash-o"></i> Delete</button>
              <a class="btn btn-default" href="<?=url("/package?id={$package->getId()}")?>"><i class="fa fa-times"></i> Cancel</a>
            </form>
          </form>
        </div>
      </div>
    </div>

  </div>
</div>

<div class="visible-xs">
  <?=block('pkg_infopanel')?>
</div>
