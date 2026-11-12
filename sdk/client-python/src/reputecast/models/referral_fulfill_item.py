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
  from ..models.referral_shipping_address import ReferralShippingAddress





T = TypeVar("T", bound="ReferralFulfillItem")



@_attrs_define
class ReferralFulfillItem:
    """ 
        Attributes:
            contact_id (str):
            email (str):
            milestone (int):
            reached_at (datetime.datetime):
            referrer (str):
            reward_id (str):
            reward_name (str):
            address (ReferralShippingAddress | Unset):
     """

    contact_id: str
    email: str
    milestone: int
    reached_at: datetime.datetime
    referrer: str
    reward_id: str
    reward_name: str
    address: ReferralShippingAddress | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_shipping_address import ReferralShippingAddress # noqa: PLC0415
        contact_id = self.contact_id

        email = self.email

        milestone = self.milestone

        reached_at = self.reached_at.isoformat()

        referrer = self.referrer

        reward_id = self.reward_id

        reward_name = self.reward_name

        address: dict[str, Any] | Unset = UNSET
        if not isinstance(self.address, Unset):
            address = self.address.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "email": email,
            "milestone": milestone,
            "reachedAt": reached_at,
            "referrer": referrer,
            "rewardId": reward_id,
            "rewardName": reward_name,
        })
        if address is not UNSET:
            field_dict["address"] = address

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_shipping_address import ReferralShippingAddress # noqa: PLC0415
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        email = d.pop("email")

        milestone = d.pop("milestone")

        reached_at = datetime.datetime.fromisoformat(d.pop("reachedAt"))




        referrer = d.pop("referrer")

        reward_id = d.pop("rewardId")

        reward_name = d.pop("rewardName")

        _address = d.pop("address", UNSET)
        address: ReferralShippingAddress | Unset
        if isinstance(_address,  Unset):
            address = UNSET
        else:
            address = ReferralShippingAddress.from_dict(_address)




        referral_fulfill_item = cls(
            contact_id=contact_id,
            email=email,
            milestone=milestone,
            reached_at=reached_at,
            referrer=referrer,
            reward_id=reward_id,
            reward_name=reward_name,
            address=address,
        )

        return referral_fulfill_item

