<?php
/* @var User $login_user */
?>
<div class="page-header">
    <h2 class="headding">GuestPass List</h2>
</div>

<div>
    <table id="app-list" class="table table-hover">

        <tr class="hidden-xs">
            <th></th>
            <th>Application Name</th>
            <th>expire date</th>
            <th>install count</th>
            <th>delete</th>
        </tr>
        <?php foreach($login_user->getGuestpasses() as $guest_pass): $app = $guest_pass->getApp();?>
        <tr>
            <td class="text-center icon">
                <a href="<?=url('/app?id='.$app->getId())?>"><img src="<?=$app->getIconUrl()?>"></a>
            </td>
            <td>
                <a class="title" href="<?=url('/app?id='.$app->getId())?>"><?=htmlspecialchars($app->getTitle())?></a>
            </td>
            <td><?=$guest_pass->getExpired()?></td>
            <td><?=$guest_pass->getInstallCount()?> installed</td>
            <td><a class="btn btn-default btn-xs" href="<?=url("/package/expire_guestpass?id={$guest_pass->getPackageId()}&guestpass_id={$guest_pass->getId()}")?>">Expire</a></td>
        </tr>
        <?php endforeach;?>
