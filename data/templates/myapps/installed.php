<div class="page-header">
  <h2 class="headding">Installed Applications</h2>
</div>

<div>
  <table id="app-list" class="table table-hover">

    <tr>
      <th></th>
     <th>title</th>
      <th>last upload</th>
      <th>notification</th>
      <th>delete</th>
    </tr>

<?php foreach($installed_apps as $ia): $app=$ia->getApp()?>
    <tr>
      <td class="text-center icon">
        <img src="<?=$app->getIconUrl()?>">
      </td>

      <td colspan="2" class="app-list-item-info">
        <div class="row">
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
      </td>

      <td class="text-center">
        <div class="btn-group notification-toggle" data-app-id="<?=$app->getId()?>">
          <button class="btn btn-default<?=$ia->getNotifySetting()?' active':''?>" value="1">ON</button>
          <button class="btn btn-default<?=$ia->getNotifySetting()?'':' active'?>" value="0">OFF</button>
        </div>
      </td>

      <td class="text-center">
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
    url: "<?=url('/api/notification_setting?id=')?>"+id+"&value="+value,
    type: "POST",
    success: function(data){
      var $btns = $('[data-app-id="'+id+'"]>button');
      $btns.removeClass('active');
      if(data.notify){
        $btns.eq(0).addClass('active');
      }
      else{
        $btns.eq(1).addClass('active');
      }
    }
  });
});

$('button.delete').on('click',function(event){
  if(confirm("このアプリケーションをインストール済みリストから削除します.\n個々のパッケージのインストール履歴は削除されません.\n削除しますか?")){
    location.href = '<?=url('/myapps/delete?id=')?>' + $(this).attr('data-app-id');
  }
});

</script>
