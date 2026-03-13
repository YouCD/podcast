import type { NotificationOptions } from "./NotificationContainer.vue";

class NotificationService {
  private container: any = null;

  /**
   * 注册通知容器
   * @param container 通知容器实例
   */
  registerContainer(container: any) {
    this.container = container;
  }

  /**
   * 显示通知
   * @param options 通知选项
   */
  notify(options: NotificationOptions) {
    if (!this.container) {
      console.warn("Notification container is not registered");
      return;
    }

    this.container.addNotification(options);
  }

  /**
   * 显示成功通知
   * @param message 消息内容
   * @param options 其他选项
   */
  success(
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
  ) {
    this.notify({
      type: "success",
      message,
      ...options,
    });
  }

  /**
   * 显示错误通知
   * @param message 消息内容
   * @param options 其他选项
   */
  error(
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
  ) {
    this.notify({
      type: "error",
      message,
      ...options,
    });
  }

  /**
   * 显示警告通知
   * @param message 消息内容
   * @param options 其他选项
   */
  warning(
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
  ) {
    this.notify({
      type: "warning",
      message,
      ...options,
    });
  }

  /**
   * 显示信息通知
   * @param message 消息内容
   * @param options 其他选项
   */
  info(
    message: string,
    options?: Omit<NotificationOptions, "message" | "type">,
  ) {
    this.notify({
      type: "info",
      message,
      ...options,
    });
  }
}

// 创建并导出单例
export const notificationService = new NotificationService();

// 默认导出
export default notificationService;
