from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.page_diff_block import PageDiffBlock
  from ..models.page_diff_setting import PageDiffSetting





T = TypeVar("T", bound="PageDiff")



@_attrs_define
class PageDiff:
    """ 
        Attributes:
            blocks (list[PageDiffBlock]):
            first (bool):
            settings (list[PageDiffSetting]):
            since (int):
            version (int):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
     """

    blocks: list[PageDiffBlock]
    first: bool
    settings: list[PageDiffSetting]
    since: int
    version: int
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_diff_block import PageDiffBlock # noqa: PLC0415
        from ..models.page_diff_setting import PageDiffSetting # noqa: PLC0415
        blocks = []
        for blocks_item_data in self.blocks:
            blocks_item = blocks_item_data.to_dict()
            blocks.append(blocks_item)



        first = self.first

        settings = []
        for settings_item_data in self.settings:
            settings_item = settings_item_data.to_dict()
            settings.append(settings_item)



        since = self.since

        version = self.version

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "first": first,
            "settings": settings,
            "since": since,
            "version": version,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_diff_block import PageDiffBlock # noqa: PLC0415
        from ..models.page_diff_setting import PageDiffSetting # noqa: PLC0415
        d = dict(src_dict)
        blocks = []
        _blocks = d.pop("blocks")
        for blocks_item_data in (_blocks):
            blocks_item = PageDiffBlock.from_dict(blocks_item_data)



            blocks.append(blocks_item)


        first = d.pop("first")

        settings = []
        _settings = d.pop("settings")
        for settings_item_data in (_settings):
            settings_item = PageDiffSetting.from_dict(settings_item_data)



            settings.append(settings_item)


        since = d.pop("since")

        version = d.pop("version")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        page_diff = cls(
            blocks=blocks,
            first=first,
            settings=settings,
            since=since,
            version=version,
            published_at=published_at,
            published_by=published_by,
        )

        return page_diff

