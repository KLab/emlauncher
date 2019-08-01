<div class="page-header">
  <h2 class="headding">Installed Applications</h2>
</div>

<div>
  <table id="app-list" class="table table-hover">

    <tr class="hidden-xs">
      <th></th>
      <th>title</th>
      <th>last upload</th>
      <th>notification</th>
      <th>delete</th>
    </tr>

<?php foreach($installed_apps as $ia): $app=$ia->getApp()?>
    <tr>
      <td class="text-center icon">
        <a href="<?=url('/app?id='.$app->getId())?>"><img src="<?=$app->getIconUrl()?>"></a>
      </td>

      <td colspan="2">
        <div class="row app-list-item-info">
          <div class="col-xs-12 col-sm-6">
            <a class="title" href="<?=url('/app?id='.$app->getId())?>"><?=htmlspecialchars($app->getTitle())?></a>
          </div>
          <div class="col-xs-12 col-sm-6">
<?php
$upload_time = $app->getLastUpload();
$update_time = $upload_time?:$app->getCreated();
?>
            <?=date('Y-m-d H:i',strtotime($update_time))?>
<?php if($login_user->getAppInstallDate($app) && $upload_time>$login_user->getAppInstallDate($app)): ?>
            <span class="label label-success">UPDATE</span>
<?php elseif(strtotime($update_time)>strtotime('yesterday')): ?>
            <span class="label label-primary">NEW</span>
<?php endif ?>
          </div>
        </div>
        <div class="row xs-buttons visible-xs">
          <b>Notification:</b>
          <div class="btn-group btn-group-sm notification-toggle" data-app-id="<?=$app->getId()?>">
            <button class="btn btn-default<?=$ia->getNotifySetting()?' active':''?>" value="1">ON</button>
            <button class="btn btn-default<?=$ia->getNotifySetting()?'':' active'?>" value="0">OFF</button>
          </div>
          <div class="pull-right container">
            <button class="btn btn-danger btn-sm delete" data-app-id="<?=$app->getId()?>"><i class="fa fa-trash-o"></i> Delete</button>
          </div>
        </div>
      </td>

      <td class="text-center hidden-xs">
        <div class="btn-group notification-toggle" data-app-id="<?=$app->getId()?>">
          <button class="btn btn-default<?=$ia->getNotifySetting()?' active':''?>" value="1">ON</button>
          <button class="btn btn-default<?=$ia->getNotifySetting()?'':' active'?>" value="0">OFF</button>
        </div>
      </td>

      <td class="text-center hidden-xs">
        <button class="btn btn-danger delete" data-app-id="<?=$app->getId()?>"><i class="fa fa-trash-o"></i></button>
      </td>
    </tr>
<?php endforeach ?>
  </table>

</div>

<script type="text/javascript">

$('.notification-toggle button').on('click',function(event){
  var id = $(this).parent().attr('data-app-id');
  var value = $(this).attr('value');
  $.ajax({
    url: "<?=url('/ajax/notification_setting?id=')?>"+id+"&value="+value,
    type: "POST",
    success: function(data){
      if(data.notify){
         $('[data-app-id="'+id+'"]>button[value="1"]').addClass('active');
         $('[data-app-id="'+id+'"]>button[value="0"]').removeClass('active');
      }
      else{
         $('[data-app-id="'+id+'"]>button[value="1"]').removeClass('active');
         $('[data-app-id="'+id+'"]>button[value="0"]').addClass('active');
      }
    }
  });
});

$('button.delete').on('click',function(event){
  if(confirm("このアプリケーションをインストール済みリストから削除します.\n個々のパッケージのインストール履歴は削除されません.\n削除しますか?")){
    location.href = '<?=url('/myapps/delete?id=')?>' + $(this).attr('data-app-id');
  }
});

$('.app-list-item-info').on('click',function(event){
  $('a',this)[0].click();
});

</script>
