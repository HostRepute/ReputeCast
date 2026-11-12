from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_composer_tier import EarnComposerTier





T = TypeVar("T", bound="EarnComposer")



@_attrs_define
class EarnComposer:
    """ 
        Attributes:
            connected (bool):
            tiers (list[EarnComposerTier]):
            upgrade_url (str):
     """

    connected: bool
    tiers: list[EarnComposerTier]
    upgrade_url: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_composer_tier import EarnComposerTier # noqa: PLC0415
        connected = self.connected

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        upgrade_url = self.upgrade_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connected": connected,
            "tiers": tiers,
            "upgradeUrl": upgrade_url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_composer_tier import EarnComposerTier # noqa: PLC0415
        d = dict(src_dict)
        connected = d.pop("connected")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnComposerTier.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        upgrade_url = d.pop("upgradeUrl")

        earn_composer = cls(
            connected=connected,
            tiers=tiers,
            upgrade_url=upgrade_url,
        )

        return earn_composer

