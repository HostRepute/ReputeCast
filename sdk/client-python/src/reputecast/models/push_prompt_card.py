from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_prompt_card_status import PushPromptCardStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.push_prompt_config import PushPromptConfig
  from ..models.push_prompt_stats import PushPromptStats





T = TypeVar("T", bound="PushPromptCard")



@_attrs_define
class PushPromptCard:
    """ 
        Attributes:
            config (PushPromptConfig):
            form_id (str):
            name (str):
            site (str):
            stats (PushPromptStats):
            status (PushPromptCardStatus):
            version (int):
     """

    config: PushPromptConfig
    form_id: str
    name: str
    site: str
    stats: PushPromptStats
    status: PushPromptCardStatus
    version: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        from ..models.push_prompt_stats import PushPromptStats # noqa: PLC0415
        config = self.config.to_dict()

        form_id = self.form_id

        name = self.name

        site = self.site

        stats = self.stats.to_dict()

        status = self.status.value

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "config": config,
            "formId": form_id,
            "name": name,
            "site": site,
            "stats": stats,
            "status": status,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_prompt_config import PushPromptConfig # noqa: PLC0415
        from ..models.push_prompt_stats import PushPromptStats # noqa: PLC0415
        d = dict(src_dict)
        config = PushPromptConfig.from_dict(d.pop("config"))




        form_id = d.pop("formId")

        name = d.pop("name")

        site = d.pop("site")

        stats = PushPromptStats.from_dict(d.pop("stats"))




        status = PushPromptCardStatus(d.pop("status"))




        version = d.pop("version")

        push_prompt_card = cls(
            config=config,
            form_id=form_id,
            name=name,
            site=site,
            stats=stats,
            status=status,
            version=version,
        )

        return push_prompt_card

