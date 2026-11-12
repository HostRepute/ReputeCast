from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.push_prompt_config import PushPromptConfig





T = TypeVar("T", bound="PushPromptSave")



@_attrs_define
class PushPromptSave:
    """ 
        Attributes:
            config (PushPromptConfig):
            site_id (str | Unset):
     """

    config: PushPromptConfig
    site_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        config = self.config.to_dict()

        site_id = self.site_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "config": config,
        })
        if site_id is not UNSET:
            field_dict["siteId"] = site_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        d = dict(src_dict)
        config = PushPromptConfig.from_dict(d.pop("config"))




        site_id = d.pop("siteId", UNSET)

        push_prompt_save = cls(
            config=config,
            site_id=site_id,
        )

        return push_prompt_save

