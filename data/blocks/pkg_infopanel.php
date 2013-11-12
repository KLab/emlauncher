<div class="list-group">
<?php if($app->isOwner($login_user)): ?>
  <div class="list-group-item">
    <ul class="nav nav-pills nav-stacked">
      <li<?=(isset($act)&&$act==='edit')?' class="active"':''?>>
        <a href="<?=url("/package/edit?id={$package->getId()}")?>"><i class="fa fa-pencil"></i> Edit</a>
      </li>
      <li>
        <a href="<?=url("/app/delete?id={$package->getId()}")?>"><i class="fa fa-trash-o"></i> Delete</a>
      </li>
    </ul>
  </div>
<?endif?>
  <div class="list-group-item">
    <div class="text-center">
      <p>link to this package</p>
      <img src="http://chart.apis.google.com/chart?chs=150&cht=qr&chl=<?=urlencode(url("/package?id={$package->getId()}"))?>">
    </div>
  </div>
</div>
