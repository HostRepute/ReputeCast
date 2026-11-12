from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_save_input import RoutingSaveInput





T = TypeVar("T", bound="RouteCampaignInput")



@_attrs_define
class RouteCampaignInput:
    """ 
        Attributes:
            draft (RoutingSaveInput | Unset):
     """

    draft: RoutingSaveInput | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_save_input import RoutingSaveInput # noqa: PLC0415
        draft: dict[str, Any] | Unset = UNSET
        if not isinstance(self.draft, Unset):
            draft = self.draft.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if draft is not UNSET:
            field_dict["draft"] = draft

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_save_input import RoutingSaveInput # noqa: PLC0415
        d = dict(src_dict)
        _draft = d.pop("draft", UNSET)
        draft: RoutingSaveInput | Unset
        if isinstance(_draft,  Unset):
            draft = UNSET
        else:
            draft = RoutingSaveInput.from_dict(_draft)




        route_campaign_input = cls(
            draft=draft,
        )

        return route_campaign_input

