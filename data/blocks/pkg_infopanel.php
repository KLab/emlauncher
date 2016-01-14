<div class="list-group">
<?php if($app->isOwner($login_user)): ?>
  <div class="list-group-item">
    <ul class="nav nav-pills nav-stacked">
      <li<?=($action==='edit')?' class="active"':''?>>
        <a href="<?=url("/package/edit?id={$package->getId()}")?>"><i class="fa fa-pencil"></i> Edit</a>
      </li>
      <li<?=($action==='delete_confirm')?' class="active"':''?>>
        <a href="<?=url("/package/delete_confirm?id={$package->getId()}")?>"><i class="fa fa-trash-o"></i> Delete</a>
      </li>
    </ul>
  </div>
<?php endif?>
  <div class="list-group-item">
    <div class="text-center">
      <p>link to this package</p>
      <img src="<?=url("/qr/code");?>?s=150&q=<?=urlencode("/package?id={$package->getId()}")?>">
    </div>
  </div>
</div>
