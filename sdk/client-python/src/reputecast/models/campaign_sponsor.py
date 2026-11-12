from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sponsor_booking_detail import SponsorBookingDetail





T = TypeVar("T", bound="CampaignSponsor")



@_attrs_define
class CampaignSponsor:
    """ 
        Attributes:
            booking (SponsorBookingDetail | Unset):
     """

    booking: SponsorBookingDetail | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_booking_detail import SponsorBookingDetail # noqa: PLC0415
        booking: dict[str, Any] | Unset = UNSET
        if not isinstance(self.booking, Unset):
            booking = self.booking.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if booking is not UNSET:
            field_dict["booking"] = booking

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_booking_detail import SponsorBookingDetail # noqa: PLC0415
        d = dict(src_dict)
        _booking = d.pop("booking", UNSET)
        booking: SponsorBookingDetail | Unset
        if isinstance(_booking,  Unset):
            booking = UNSET
        else:
            booking = SponsorBookingDetail.from_dict(_booking)




        campaign_sponsor = cls(
            booking=booking,
        )

        return campaign_sponsor

