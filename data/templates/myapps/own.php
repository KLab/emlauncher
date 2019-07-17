<div class="page-header">
  <h2 class="headding">Own Applications</h2>
</div>

<div class="row">
<?php $counter = 0; ?>
<?php foreach($own_apps as $app):?>
<div class="media app-list-item col-md-6">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon-sm media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h3 class="media-hedding">
      <a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a>
      <small title="Installed by <?=$app->getInstallUserCount()?> users" class="badge"><?=$app->getInstallUserCount()?></small>
    </h3>
    <p>
<?php
$upload_time = $app->getLastUpload();
$update_time = $upload_time?:$app->getCreated();
?>
      <?=($upload_time)?'last uploaded':'created'?>: <?=date('Y-m-d H:i',strtotime($update_time))?>
<?php if($login_user->getAppInstallDate($app) && $upload_time>$login_user->getAppInstallDate($app)): ?>
      <span class="label label-success">UPDATE</span>
<?php elseif(strtotime($update_time)>strtotime('yesterday')): ?>
      <span class="label label-primary">NEW</span>
<?php endif ?>
    </p>
  </div>
</div>
<?php if((++$counter)%2===0): ?>
</div>
<div class="row">
<?php endif ?>
<?php endforeach ?>
</div>

<script type="text/javascript">

$('.notification-toggle button').on('click',function(event){
  var id = $(this).parent().attr('data-app-id');
  var value = $(this).attr('value');
  $.ajax({
    url: "<?=url('/api/notification_setting?id=')?>"+id+"&value="+value,
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
