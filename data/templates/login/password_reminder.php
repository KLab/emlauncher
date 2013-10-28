
<div id="reminder-dialog" class="col-sm-6 col-sm-offset-3">
  <div class="panel panel-primary">
    <div class="panel-heading">
      <h2 class="panel-title">Reset password</h2>
    </div>

    <div class="panel-body">
      <form class="form-horizontal" method="post" action="<?=url('/login/password_confirm')?>">

        <div class="form-group">
          <label class="control-label col-sm-3" for="email">email</label>
          <div class="col-sm-9">
            <input class="form-control" type="text" id="email" name="email">
            <span class="help-block">このアドレスにリセットURLを送信します</span>
          </div>
        </div>

        <div class="col-sm-9 col-sm-offset-3">
          <input type="submit" class="btn btn-primary" value="送信">
        </div>
      </form>

    </div>
  </div>
</div>