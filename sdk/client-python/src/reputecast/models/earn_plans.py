from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.brand import Brand
  from ..models.earn_free_card import EarnFreeCard
  from ..models.earn_list_ref import EarnListRef
  from ..models.earn_tier import EarnTier
  from ..models.earn_upgrade_card import EarnUpgradeCard





T = TypeVar("T", bound="EarnPlans")



@_attrs_define
class EarnPlans:
    """ 
        Attributes:
            brand (Brand):
            can_edit (bool):
            connected (bool):
            currency (str):
            free (EarnFreeCard):
            helpers (list[str]):
            lists (list[EarnListRef]):
            tiers (list[EarnTier]):
            upgrade_page (EarnUpgradeCard):
     """

    brand: Brand
    can_edit: bool
    connected: bool
    currency: str
    free: EarnFreeCard
    helpers: list[str]
    lists: list[EarnListRef]
    tiers: list[EarnTier]
    upgrade_page: EarnUpgradeCard





    def to_dict(self) -> dict[str, Any]:
        from ..models.brand import Brand # noqa: PLC0415
        from ..models.earn_free_card import EarnFreeCard # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_tier import EarnTier # noqa: PLC0415
        from ..models.earn_upgrade_card import EarnUpgradeCard # noqa: PLC0415
        brand = self.brand.to_dict()

        can_edit = self.can_edit

        connected = self.connected

        currency = self.currency

        free = self.free.to_dict()

        helpers = self.helpers



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        upgrade_page = self.upgrade_page.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "brand": brand,
            "canEdit": can_edit,
            "connected": connected,
            "currency": currency,
            "free": free,
            "helpers": helpers,
            "lists": lists,
            "tiers": tiers,
            "upgradePage": upgrade_page,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.brand import Brand # noqa: PLC0415
        from ..models.earn_free_card import EarnFreeCard # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        from ..models.earn_tier import EarnTier # noqa: PLC0415
        from ..models.earn_upgrade_card import EarnUpgradeCard # noqa: PLC0415
        d = dict(src_dict)
        brand = Brand.from_dict(d.pop("brand"))




        can_edit = d.pop("canEdit")

        connected = d.pop("connected")

        currency = d.pop("currency")

        free = EarnFreeCard.from_dict(d.pop("free"))




        helpers = cast(list[str], d.pop("helpers"))


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = EarnListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        upgrade_page = EarnUpgradeCard.from_dict(d.pop("upgradePage"))




        earn_plans = cls(
            brand=brand,
            can_edit=can_edit,
            connected=connected,
            currency=currency,
            free=free,
            helpers=helpers,
            lists=lists,
            tiers=tiers,
            upgrade_page=upgrade_page,
        )

        return earn_plans

