from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.referral_quality_line import ReferralQualityLine





T = TypeVar("T", bound="ReferralQuality")



@_attrs_define
class ReferralQuality:
    """ 
        Attributes:
            bounced (ReferralQualityLine):
            campaign (str):
            enough (bool):
            opens (ReferralQualityLine):
            referred (int):
            unsubscribed (ReferralQualityLine):
     """

    bounced: ReferralQualityLine
    campaign: str
    enough: bool
    opens: ReferralQualityLine
    referred: int
    unsubscribed: ReferralQualityLine





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_quality_line import ReferralQualityLine # noqa: PLC0415
        bounced = self.bounced.to_dict()

        campaign = self.campaign

        enough = self.enough

        opens = self.opens.to_dict()

        referred = self.referred

        unsubscribed = self.unsubscribed.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounced": bounced,
            "campaign": campaign,
            "enough": enough,
            "opens": opens,
            "referred": referred,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_quality_line import ReferralQualityLine # noqa: PLC0415
        d = dict(src_dict)
        bounced = ReferralQualityLine.from_dict(d.pop("bounced"))




        campaign = d.pop("campaign")

        enough = d.pop("enough")

        opens = ReferralQualityLine.from_dict(d.pop("opens"))




        referred = d.pop("referred")

        unsubscribed = ReferralQualityLine.from_dict(d.pop("unsubscribed"))




        referral_quality = cls(
            bounced=bounced,
            campaign=campaign,
            enough=enough,
            opens=opens,
            referred=referred,
            unsubscribed=unsubscribed,
        )

        return referral_quality

