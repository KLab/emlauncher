
<div id="reminder-dialog" class="col-sm-6 col-sm-offset-3">
<?php if($error): ?>
  <div class="panel panel-danger">
    <div class="panel-heading">
      <h2 class="panel-title">送信できませんでした</h2>
    </div>
    <div class="panel-body">
      <p>登録されたメールアドレスが正しく入力されているか確認してください。</p>
    </div>
  </div>

<?php else: ?>
  <div class="panel panel-success">
    <div class="panel-heading">
      <h2 class="panel-title">メールを送信しました</h2>
    </div>
    <div class="panel-body">
      <p>メールに記載のURLにアクセスし、新しいパスワードを設定してください。</p>
    </div>
  </div>
<?php endif ?>
</div>

