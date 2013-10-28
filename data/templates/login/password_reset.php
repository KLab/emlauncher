
<div class="col-sm-6 col-sm-offset-3">
  <div class="panel panel-primary">
    <div class="panel-heading">
      <h2 class="panel-title">Reset Password</h2>
    </div>

    <div class="panel-body">
      <div id="alert_nopassword" class="alert alert-danger hidden">
        パスワードが入力されていません
      </div>
      <div id="alert_missmatch" class="alert alert-danger hidden">
        パスワードが一致しません
      </div>

      <form class="form-horizontal" method="post" action="<?=url('/login/password_commit')?>">
        <input type="hidden" name="key" value="<?=$key?>">

        <div class="form-group">
          <label class="control-label col-sm-3" for="password">password</label>
          <div class="col-sm-9">
            <input class="form-control" type="password" id="password" name="password">
          </div>
        </div>

        <div class="form-group">
          <label class="control-label col-sm-3" for="confirm">confirm</label>
          <div class="col-sm-9">
            <input class="form-control" type="password" id="confirm" name="confirm">
          </div>
        </div>

        <div class="col-sm-9 col-sm-offset-3">
          <input type="submit" class="btn btn-primary" value="reset password">
        </div>
      </form>
    </div>
  </div>
</div>
<script type="text/javascript">
$('form').on('submit',function(){
  var pass1 = $('input[name="password"]',this).val();
  var pass2 = $('input[name="confirm"]',this).val();
  $('.alert').addClass('hidden');
  if(pass1=='' || pass2==''){
    $('#alert_nopassword').removeClass('hidden');
    return false;
  }
  if(pass1!=pass2){
    $('#alert_missmatch').removeClass('hidden');
    return false;
  }
  return true;
});
</script>
