from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_audience_left_out import EarnAudienceLeftOut
  from ..models.earn_audience_tier import EarnAudienceTier





T = TypeVar("T", bound="EarnAudience")



@_attrs_define
class EarnAudience:
    """ 
        Attributes:
            complimentary (int):
            connected (bool):
            everyone (int):
            free (int):
            gift (int):
            left_out (EarnAudienceLeftOut):
            paid (int):
            past_due (int):
            tiers (list[EarnAudienceTier]):
            trialing (int):
            with_access (int):
     """

    complimentary: int
    connected: bool
    everyone: int
    free: int
    gift: int
    left_out: EarnAudienceLeftOut
    paid: int
    past_due: int
    tiers: list[EarnAudienceTier]
    trialing: int
    with_access: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_audience_left_out import EarnAudienceLeftOut # noqa: PLC0415
        from ..models.earn_audience_tier import EarnAudienceTier # noqa: PLC0415
        complimentary = self.complimentary

        connected = self.connected

        everyone = self.everyone

        free = self.free

        gift = self.gift

        left_out = self.left_out.to_dict()

        paid = self.paid

        past_due = self.past_due

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        trialing = self.trialing

        with_access = self.with_access


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "complimentary": complimentary,
            "connected": connected,
            "everyone": everyone,
            "free": free,
            "gift": gift,
            "leftOut": left_out,
            "paid": paid,
            "pastDue": past_due,
            "tiers": tiers,
            "trialing": trialing,
            "withAccess": with_access,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_audience_left_out import EarnAudienceLeftOut # noqa: PLC0415
        from ..models.earn_audience_tier import EarnAudienceTier # noqa: PLC0415
        d = dict(src_dict)
        complimentary = d.pop("complimentary")

        connected = d.pop("connected")

        everyone = d.pop("everyone")

        free = d.pop("free")

        gift = d.pop("gift")

        left_out = EarnAudienceLeftOut.from_dict(d.pop("leftOut"))




        paid = d.pop("paid")

        past_due = d.pop("pastDue")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnAudienceTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        trialing = d.pop("trialing")

        with_access = d.pop("withAccess")

        earn_audience = cls(
            complimentary=complimentary,
            connected=connected,
            everyone=everyone,
            free=free,
            gift=gift,
            left_out=left_out,
            paid=paid,
            past_due=past_due,
            tiers=tiers,
            trialing=trialing,
            with_access=with_access,
        )

        return earn_audience

