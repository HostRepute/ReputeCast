from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_composer_offer import EarnComposerOffer





T = TypeVar("T", bound="EarnComposerTier")



@_attrs_define
class EarnComposerTier:
    """ 
        Attributes:
            id (str):
            includes (list[str]):
            name (str):
            offers (list[EarnComposerOffer]):
            price_line (str):
            short (str):
            top (bool):
            trial_days (int):
     """

    id: str
    includes: list[str]
    name: str
    offers: list[EarnComposerOffer]
    price_line: str
    short: str
    top: bool
    trial_days: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_composer_offer import EarnComposerOffer # noqa: PLC0415
        id = self.id

        includes = self.includes



        name = self.name

        offers = []
        for offers_item_data in self.offers:
            offers_item = offers_item_data.to_dict()
            offers.append(offers_item)



        price_line = self.price_line

        short = self.short

        top = self.top

        trial_days = self.trial_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "includes": includes,
            "name": name,
            "offers": offers,
            "priceLine": price_line,
            "short": short,
            "top": top,
            "trialDays": trial_days,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_composer_offer import EarnComposerOffer # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        includes = cast(list[str], d.pop("includes"))


        name = d.pop("name")

        offers = []
        _offers = d.pop("offers")
        for offers_item_data in (_offers):
            offers_item = EarnComposerOffer.from_dict(offers_item_data)



            offers.append(offers_item)


        price_line = d.pop("priceLine")

        short = d.pop("short")

        top = d.pop("top")

        trial_days = d.pop("trialDays")

        earn_composer_tier = cls(
            id=id,
            includes=includes,
            name=name,
            offers=offers,
            price_line=price_line,
            short=short,
            top=top,
            trial_days=trial_days,
        )

        return earn_composer_tier

