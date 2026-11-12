from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_sender_option import TransactionalSenderOption
  from ..models.transactional_sender_ref import TransactionalSenderRef





T = TypeVar("T", bound="TransactionalSettings")



@_attrs_define
class TransactionalSettings:
    """ 
        Attributes:
            alert_bounce_rate (float): Percent of a template's messages in an hour
            alert_failed (int): Failed messages in an hour
            alert_median_seconds (int): Median delivery time over an hour
            can_edit (bool):
            content_retention_days (int):
            default_reply_to (str):
            default_track_clicks (bool):
            default_track_opens (bool):
            link_contacts (bool): Link recipients who match a contact to that contact (never creates contacts)
            product_name (str):
            senders (list[TransactionalSenderOption]):
            default_sender (TransactionalSenderRef | Unset):
            reason (str | Unset):
            updated_at (datetime.datetime | Unset):
            updated_by (str | Unset):
     """

    alert_bounce_rate: float
    alert_failed: int
    alert_median_seconds: int
    can_edit: bool
    content_retention_days: int
    default_reply_to: str
    default_track_clicks: bool
    default_track_opens: bool
    link_contacts: bool
    product_name: str
    senders: list[TransactionalSenderOption]
    default_sender: TransactionalSenderRef | Unset = UNSET
    reason: str | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_sender_option import TransactionalSenderOption # noqa: PLC0415
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        alert_bounce_rate = self.alert_bounce_rate

        alert_failed = self.alert_failed

        alert_median_seconds = self.alert_median_seconds

        can_edit = self.can_edit

        content_retention_days = self.content_retention_days

        default_reply_to = self.default_reply_to

        default_track_clicks = self.default_track_clicks

        default_track_opens = self.default_track_opens

        link_contacts = self.link_contacts

        product_name = self.product_name

        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        default_sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.default_sender, Unset):
            default_sender = self.default_sender.to_dict()

        reason = self.reason

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertBounceRate": alert_bounce_rate,
            "alertFailed": alert_failed,
            "alertMedianSeconds": alert_median_seconds,
            "canEdit": can_edit,
            "contentRetentionDays": content_retention_days,
            "defaultReplyTo": default_reply_to,
            "defaultTrackClicks": default_track_clicks,
            "defaultTrackOpens": default_track_opens,
            "linkContacts": link_contacts,
            "productName": product_name,
            "senders": senders,
        })
        if default_sender is not UNSET:
            field_dict["defaultSender"] = default_sender
        if reason is not UNSET:
            field_dict["reason"] = reason
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_sender_option import TransactionalSenderOption # noqa: PLC0415
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        d = dict(src_dict)
        alert_bounce_rate = d.pop("alertBounceRate")

        alert_failed = d.pop("alertFailed")

        alert_median_seconds = d.pop("alertMedianSeconds")

        can_edit = d.pop("canEdit")

        content_retention_days = d.pop("contentRetentionDays")

        default_reply_to = d.pop("defaultReplyTo")

        default_track_clicks = d.pop("defaultTrackClicks")

        default_track_opens = d.pop("defaultTrackOpens")

        link_contacts = d.pop("linkContacts")

        product_name = d.pop("productName")

        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = TransactionalSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        _default_sender = d.pop("defaultSender", UNSET)
        default_sender: TransactionalSenderRef | Unset
        if isinstance(_default_sender,  Unset):
            default_sender = UNSET
        else:
            default_sender = TransactionalSenderRef.from_dict(_default_sender)




        reason = d.pop("reason", UNSET)

        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        updated_by = d.pop("updatedBy", UNSET)

        transactional_settings = cls(
            alert_bounce_rate=alert_bounce_rate,
            alert_failed=alert_failed,
            alert_median_seconds=alert_median_seconds,
            can_edit=can_edit,
            content_retention_days=content_retention_days,
            default_reply_to=default_reply_to,
            default_track_clicks=default_track_clicks,
            default_track_opens=default_track_opens,
            link_contacts=link_contacts,
            product_name=product_name,
            senders=senders,
            default_sender=default_sender,
            reason=reason,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return transactional_settings

