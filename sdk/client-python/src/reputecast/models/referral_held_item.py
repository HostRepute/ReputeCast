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
  from ..models.referral_engagement import ReferralEngagement
  from ..models.referral_network import ReferralNetwork





T = TypeVar("T", bound="ReferralHeldItem")



@_attrs_define
class ReferralHeldItem:
    """ 
        Attributes:
            email (str):
            engagement (ReferralEngagement):
            id (str):
            ip (str):
            network (ReferralNetwork):
            signals (list[str]):
            signed_up_at (datetime.datetime):
            confirmed_at (datetime.datetime | Unset):
            hold_ends (datetime.datetime | Unset):
     """

    email: str
    engagement: ReferralEngagement
    id: str
    ip: str
    network: ReferralNetwork
    signals: list[str]
    signed_up_at: datetime.datetime
    confirmed_at: datetime.datetime | Unset = UNSET
    hold_ends: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_engagement import ReferralEngagement # noqa: PLC0415
        from ..models.referral_network import ReferralNetwork # noqa: PLC0415
        email = self.email

        engagement = self.engagement.to_dict()

        id = self.id

        ip = self.ip

        network = self.network.to_dict()

        signals = self.signals



        signed_up_at = self.signed_up_at.isoformat()

        confirmed_at: str | Unset = UNSET
        if not isinstance(self.confirmed_at, Unset):
            confirmed_at = self.confirmed_at.isoformat()

        hold_ends: str | Unset = UNSET
        if not isinstance(self.hold_ends, Unset):
            hold_ends = self.hold_ends.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "engagement": engagement,
            "id": id,
            "ip": ip,
            "network": network,
            "signals": signals,
            "signedUpAt": signed_up_at,
        })
        if confirmed_at is not UNSET:
            field_dict["confirmedAt"] = confirmed_at
        if hold_ends is not UNSET:
            field_dict["holdEnds"] = hold_ends

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_engagement import ReferralEngagement # noqa: PLC0415
        from ..models.referral_network import ReferralNetwork # noqa: PLC0415
        d = dict(src_dict)
        email = d.pop("email")

        engagement = ReferralEngagement.from_dict(d.pop("engagement"))




        id = d.pop("id")

        ip = d.pop("ip")

        network = ReferralNetwork.from_dict(d.pop("network"))




        signals = cast(list[str], d.pop("signals"))


        signed_up_at = datetime.datetime.fromisoformat(d.pop("signedUpAt"))




        _confirmed_at = d.pop("confirmedAt", UNSET)
        confirmed_at: datetime.datetime | Unset
        if isinstance(_confirmed_at,  Unset):
            confirmed_at = UNSET
        else:
            confirmed_at = datetime.datetime.fromisoformat(_confirmed_at)




        _hold_ends = d.pop("holdEnds", UNSET)
        hold_ends: datetime.datetime | Unset
        if isinstance(_hold_ends,  Unset):
            hold_ends = UNSET
        else:
            hold_ends = datetime.datetime.fromisoformat(_hold_ends)




        referral_held_item = cls(
            email=email,
            engagement=engagement,
            id=id,
            ip=ip,
            network=network,
            signals=signals,
            signed_up_at=signed_up_at,
            confirmed_at=confirmed_at,
            hold_ends=hold_ends,
        )

        return referral_held_item

