<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
	  <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><?=htmlspecialchars($app->getTitle())?></h2>
    <p><?=htmlspecialchars($app->getDescription())?></p>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('app_infopanel')?>
  </div>

  <div class="col-xs-12 col-sm-8 col-md-9">

    <div class="well">
      <form id="refresh-apikey" class="form-inline" method="post" action="<?=url('/app/preference_refresh_apikey')?>">
        <legend>API Key</legend>
        <input type="hidden" name="id" value="<?=$app->getId()?>">
        <div class="form-group">
          <label class="sr-only" for="api-key">API Key</label>
          <input type="text" id="api-key" name="api_key" class="form-control" readonly="readonly" value="<?=htmlspecialchars($app->getAPIKey())?>">
        </div>
        <button id="submit-refresh-apikey" type="submit" class="btn btn-warning"><i class="fa fa-refresh"></i> Refresh</button>
        <div class="help-block">
          Upload APIを利用するために必要なキーです.
          詳細は<a href="#">APIドキュメント</a>を参照してください.
        </div>
      </form>
    </div>

    <div class="well">
      <form id="edit-info" class="form-horizontal" method="post" action="<?=url('/app/preference_update')?>" enctype="multipart/form-data">
        <legend>Edit Informations</legend>

        <div class="row">
          <div class="col-md-9 col-xs-12">
            <div class="form-group">
	          <label for="title" class="control-label col-md-2 required">Title</label>
              <div class="col-md-10">
                <div id="alert-notitle" class="alert alert-danger hidden">
                  タイトルが入力されていません
                </div>
                <input class="form-control" type="text" id="title" name="title" value="<?=htmlspecialchars($app->getTitle())?>">
              </div>
            </div>

            <div class="form-group">
              <label for="icon-selector" class="control-label col-md-2 required">Icon</label>
              <div class="col-md-10">
                <div id="alert-icon-size-limit" class="alert alert-danger hidden">
                  画像ファイルサイズが大きすぎます
                </div>
                <input type="hidden" id="icon-data" name="icon-data" value="">
                <input type="file" id="icon-selector" class="hidden">
                <div class="input-group"  id="input-group-icon">
                  <input type="text" class="form-control droparea" id="icon-text" readonly="readonly">
                  <a id="icon-browse" class="input-group-addon btn btn-default">Browse</a>
                </div>
              </div>
            </div>
          </div>

          <div class="col-md-3 hidden-sm hidden-xs text-center">
            <img id="icon-preview" class="img-thumbnail droparea" style="width:96px;height:96px;" src="<?=$app->getIconUrl()?>">
          </div>
        </div>

        <div class="row">
          <div class="col-md-9 col-xs-12">
            <div class="form-group">
              <label for="description" class="control-label col-md-2">Description</label>
              <div class="col-md-10">
                <textarea class="form-control" row="3" id="description" name="description"><?=htmlspecialchars($app->getDescription())?></textarea>
              </div>
            </div>
          </div>
        </div>

        <div class="row">
          <div class="col-md-9 col-xs-12">
            <div class="form-group">
              <label for="repository" class="control-label col-md-2">Repository</label>
              <div class="col-md-10">
                <input type="text" class="form-control" id="repository" name="repository">
              </div>
            </div>
          </div>
        </div>

        <div class="row">
          <div class="col-md-9 col-xs-12">
            <div class="form-group">
              <div class="col-md-10 col-md-offset-2">
                <button class="btn btn-primary"><i class="fa fa-save"></i> Update</button>
              </div>
            </div>
          </div>
        </div>
      </form>
    </div>

    <div class="well">
      <form id="edit-tags" class="form-horizontal">
        <legend>Delete Tags</legend>

        <div class="form-group">
          <div class="col-xs-12">
<?php foreach($app->getTags() as $tag): ?>
            <input type="checkbox" class="hidden" name="tags[]" value="<?=htmlspecialchars($tag->getName())?>">
            <button class="btn btn-default delete-tags" data-toggle="button"><?=htmlspecialchars($tag->getName())?></button>
<?php endforeach ?>
          </div>
        </div>

        <div class="form-group">
          <div class="col-xs-12">
          <button class="btn btn-danger"><i class="fa fa-save"></i> Delete</button>
          </div>
        </div>

      </form>
    </div>
  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

<script type="text/javascript">

$('#submit-refresh-apikey').on('click',function(){
  return confirm('現在のAPI Keyは使用できなくなります。\nよろしいですか？');
});

</script>
