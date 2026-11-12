from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_render_input_variant import CampaignRenderInputVariant
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="CampaignRenderInput")



@_attrs_define
class CampaignRenderInput:
    """ 
        Attributes:
            clients (list[str]): Client ids from the connection's clients; each is one preview on the account's plan
            variant (CampaignRenderInputVariant | Unset): The A/B variant to render (A by default)
     """

    clients: list[str]
    variant: CampaignRenderInputVariant | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clients = self.clients



        variant: str | Unset = UNSET
        if not isinstance(self.variant, Unset):
            variant = self.variant.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clients": clients,
        })
        if variant is not UNSET:
            field_dict["variant"] = variant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clients = cast(list[str], d.pop("clients"))


        _variant = d.pop("variant", UNSET)
        variant: CampaignRenderInputVariant | Unset
        if isinstance(_variant,  Unset):
            variant = UNSET
        else:
            variant = CampaignRenderInputVariant(_variant)




        campaign_render_input = cls(
            clients=clients,
            variant=variant,
        )

        return campaign_render_input

