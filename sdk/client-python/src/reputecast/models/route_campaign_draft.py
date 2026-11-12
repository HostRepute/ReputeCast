from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.route_campaign_share import RouteCampaignShare





T = TypeVar("T", bound="RouteCampaignDraft")



@_attrs_define
class RouteCampaignDraft:
    """ 
        Attributes:
            moved (int):
            shares (list[RouteCampaignShare]):
     """

    moved: int
    shares: list[RouteCampaignShare]





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_campaign_share import RouteCampaignShare # noqa: PLC0415
        moved = self.moved

        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "moved": moved,
            "shares": shares,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_campaign_share import RouteCampaignShare # noqa: PLC0415
        d = dict(src_dict)
        moved = d.pop("moved")

        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = RouteCampaignShare.from_dict(shares_item_data)



            shares.append(shares_item)


        route_campaign_draft = cls(
            moved=moved,
            shares=shares,
        )

        return route_campaign_draft

