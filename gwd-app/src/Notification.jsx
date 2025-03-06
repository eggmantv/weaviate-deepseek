/* eslint-disable react-refresh/only-export-components */
import toast from "react-hot-toast";
import i18next from "i18next";

const notificationStyle = {
  borderRadius: "24px",
  padding: "8px 16px", // x, y
};

export const clearedNotify = () => {
  toast.success(i18next.t("notification.cleared"), {
    style: notificationStyle,
  });
};

export const copiedNotify = () => {
  toast.success(i18next.t("notification.copied"), {
    style: notificationStyle,
  });
};

export const resetNotify = () => {
  toast.success(i18next.t("notification.reset"), {
    style: notificationStyle,
  });
};

export const NotificationError = (error) => {
  toast.error(error, { style: notificationStyle });
};

export const NotifictionSuccess = (data) => {
  toast.success(data, {
    style: notificationStyle,
  });
};

export const networkErrorNotify = () => {
  toast.error(i18next.t("notification.network-error"), {
    style: notificationStyle,
  });
};

export const deletedNotify = () => {
  toast.success(i18next.t("notification.deleted"), {
    style: notificationStyle,
  });
};

export const changeNotify = () => {
  toast.error(
    <span>
      成为会员体验更多功能
      <a
        style={{ marginLeft: 10, color: "#b07807" }}
        target="_blank"
        rel="noopener noreferrer"
        href="https://eggman.tv/subscriptions"
      >
        订阅
      </a>
    </span>,
    {
      style: notificationStyle,
    }
  );
};
