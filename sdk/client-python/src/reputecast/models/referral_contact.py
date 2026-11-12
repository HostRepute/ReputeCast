from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.referral_contact_referred_by import ReferralContactReferredBy
  from ..models.referral_contact_referrer import ReferralContactReferrer
  from ..models.referral_timeline_event import ReferralTimelineEvent





T = TypeVar("T", bound="ReferralContact")



@_attrs_define
class ReferralContact:
    """ 
        Attributes:
            events (list[ReferralTimelineEvent]):
            referred_by (ReferralContactReferredBy | Unset):
            referrer (ReferralContactReferrer | Unset):
     """

    events: list[ReferralTimelineEvent]
    referred_by: ReferralContactReferredBy | Unset = UNSET
    referrer: ReferralContactReferrer | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_contact_referred_by import ReferralContactReferredBy # noqa: PLC0415
        from ..models.referral_contact_referrer import ReferralContactReferrer # noqa: PLC0415
        from ..models.referral_timeline_event import ReferralTimelineEvent # noqa: PLC0415
        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        referred_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.referred_by, Unset):
            referred_by = self.referred_by.to_dict()

        referrer: dict[str, Any] | Unset = UNSET
        if not isinstance(self.referrer, Unset):
            referrer = self.referrer.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
        })
        if referred_by is not UNSET:
            field_dict["referredBy"] = referred_by
        if referrer is not UNSET:
            field_dict["referrer"] = referrer

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_contact_referred_by import ReferralContactReferredBy # noqa: PLC0415
        from ..models.referral_contact_referrer import ReferralContactReferrer # noqa: PLC0415
        from ..models.referral_timeline_event import ReferralTimelineEvent # noqa: PLC0415
        d = dict(src_dict)
        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = ReferralTimelineEvent.from_dict(events_item_data)



            events.append(events_item)


        _referred_by = d.pop("referredBy", UNSET)
        referred_by: ReferralContactReferredBy | Unset
        if isinstance(_referred_by,  Unset):
            referred_by = UNSET
        else:
            referred_by = ReferralContactReferredBy.from_dict(_referred_by)




        _referrer = d.pop("referrer", UNSET)
        referrer: ReferralContactReferrer | Unset
        if isinstance(_referrer,  Unset):
            referrer = UNSET
        else:
            referrer = ReferralContactReferrer.from_dict(_referrer)




        referral_contact = cls(
            events=events,
            referred_by=referred_by,
            referrer=referrer,
        )

        return referral_contact

