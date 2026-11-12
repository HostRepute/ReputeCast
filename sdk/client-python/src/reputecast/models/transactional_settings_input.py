from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalSettingsInput")



@_attrs_define
class TransactionalSettingsInput:
    """ 
        Attributes:
            alert_bounce_rate (float):
            alert_failed (int):
            alert_median_seconds (int):
            content_retention_days (int):
            default_track_clicks (bool):
            default_track_opens (bool):
            link_contacts (bool):
            default_reply_to (str | Unset):
            default_sender_id (str | Unset): A Transactional sender, or empty
     """

    alert_bounce_rate: float
    alert_failed: int
    alert_median_seconds: int
    content_retention_days: int
    default_track_clicks: bool
    default_track_opens: bool
    link_contacts: bool
    default_reply_to: str | Unset = UNSET
    default_sender_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        alert_bounce_rate = self.alert_bounce_rate

        alert_failed = self.alert_failed

        alert_median_seconds = self.alert_median_seconds

        content_retention_days = self.content_retention_days

        default_track_clicks = self.default_track_clicks

        default_track_opens = self.default_track_opens

        link_contacts = self.link_contacts

        default_reply_to = self.default_reply_to

        default_sender_id = self.default_sender_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertBounceRate": alert_bounce_rate,
            "alertFailed": alert_failed,
            "alertMedianSeconds": alert_median_seconds,
            "contentRetentionDays": content_retention_days,
            "defaultTrackClicks": default_track_clicks,
            "defaultTrackOpens": default_track_opens,
            "linkContacts": link_contacts,
        })
        if default_reply_to is not UNSET:
            field_dict["defaultReplyTo"] = default_reply_to
        if default_sender_id is not UNSET:
            field_dict["defaultSenderId"] = default_sender_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alert_bounce_rate = d.pop("alertBounceRate")

        alert_failed = d.pop("alertFailed")

        alert_median_seconds = d.pop("alertMedianSeconds")

        content_retention_days = d.pop("contentRetentionDays")

        default_track_clicks = d.pop("defaultTrackClicks")

        default_track_opens = d.pop("defaultTrackOpens")

        link_contacts = d.pop("linkContacts")

        default_reply_to = d.pop("defaultReplyTo", UNSET)

        default_sender_id = d.pop("defaultSenderId", UNSET)

        transactional_settings_input = cls(
            alert_bounce_rate=alert_bounce_rate,
            alert_failed=alert_failed,
            alert_median_seconds=alert_median_seconds,
            content_retention_days=content_retention_days,
            default_track_clicks=default_track_clicks,
            default_track_opens=default_track_opens,
            link_contacts=link_contacts,
            default_reply_to=default_reply_to,
            default_sender_id=default_sender_id,
        )

        return transactional_settings_input

