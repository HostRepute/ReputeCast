from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_receiving_mode import LeadAdsReceivingMode
from ..models.lead_ads_receiving_reason import LeadAdsReceivingReason
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="LeadAdsReceiving")



@_attrs_define
class LeadAdsReceiving:
    """ 
        Attributes:
            mode (LeadAdsReceivingMode):
            url (str): The instance's public URL
            last_event_at (datetime.datetime | Unset): Meta's last delivery
            last_lead_at (datetime.datetime | Unset):
            last_polled_at (datetime.datetime | Unset):
            reason (LeadAdsReceivingReason | Unset):
            subscribed_at (datetime.datetime | Unset): When Meta accepted the app's webhook subscription
     """

    mode: LeadAdsReceivingMode
    url: str
    last_event_at: datetime.datetime | Unset = UNSET
    last_lead_at: datetime.datetime | Unset = UNSET
    last_polled_at: datetime.datetime | Unset = UNSET
    reason: LeadAdsReceivingReason | Unset = UNSET
    subscribed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        url = self.url

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        last_lead_at: str | Unset = UNSET
        if not isinstance(self.last_lead_at, Unset):
            last_lead_at = self.last_lead_at.isoformat()

        last_polled_at: str | Unset = UNSET
        if not isinstance(self.last_polled_at, Unset):
            last_polled_at = self.last_polled_at.isoformat()

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        subscribed_at: str | Unset = UNSET
        if not isinstance(self.subscribed_at, Unset):
            subscribed_at = self.subscribed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "url": url,
        })
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if last_lead_at is not UNSET:
            field_dict["lastLeadAt"] = last_lead_at
        if last_polled_at is not UNSET:
            field_dict["lastPolledAt"] = last_polled_at
        if reason is not UNSET:
            field_dict["reason"] = reason
        if subscribed_at is not UNSET:
            field_dict["subscribedAt"] = subscribed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = LeadAdsReceivingMode(d.pop("mode"))




        url = d.pop("url")

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _last_lead_at = d.pop("lastLeadAt", UNSET)
        last_lead_at: datetime.datetime | Unset
        if isinstance(_last_lead_at,  Unset):
            last_lead_at = UNSET
        else:
            last_lead_at = datetime.datetime.fromisoformat(_last_lead_at)




        _last_polled_at = d.pop("lastPolledAt", UNSET)
        last_polled_at: datetime.datetime | Unset
        if isinstance(_last_polled_at,  Unset):
            last_polled_at = UNSET
        else:
            last_polled_at = datetime.datetime.fromisoformat(_last_polled_at)




        _reason = d.pop("reason", UNSET)
        reason: LeadAdsReceivingReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = LeadAdsReceivingReason(_reason)




        _subscribed_at = d.pop("subscribedAt", UNSET)
        subscribed_at: datetime.datetime | Unset
        if isinstance(_subscribed_at,  Unset):
            subscribed_at = UNSET
        else:
            subscribed_at = datetime.datetime.fromisoformat(_subscribed_at)




        lead_ads_receiving = cls(
            mode=mode,
            url=url,
            last_event_at=last_event_at,
            last_lead_at=last_lead_at,
            last_polled_at=last_polled_at,
            reason=reason,
            subscribed_at=subscribed_at,
        )

        return lead_ads_receiving

