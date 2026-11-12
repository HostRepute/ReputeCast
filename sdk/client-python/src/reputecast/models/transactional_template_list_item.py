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
  from ..models.transactional_category_ref import TransactionalCategoryRef
  from ..models.transactional_stats import TransactionalStats





T = TypeVar("T", bound="TransactionalTemplateListItem")



@_attrs_define
class TransactionalTemplateListItem:
    """ 
        Attributes:
            archived (bool):
            attachments (bool): Recent messages carried attachments
            description (str):
            draft_changes (bool):
            draft_version (int):
            id (str):
            key (str):
            name (str):
            spark (list[int]):
            stats (TransactionalStats):
            updated_at (datetime.datetime):
            category (TransactionalCategoryRef | Unset):
            last_sent_at (datetime.datetime | Unset):
            live_version (int | Unset):
            used_by_products (int | Unset):
     """

    archived: bool
    attachments: bool
    description: str
    draft_changes: bool
    draft_version: int
    id: str
    key: str
    name: str
    spark: list[int]
    stats: TransactionalStats
    updated_at: datetime.datetime
    category: TransactionalCategoryRef | Unset = UNSET
    last_sent_at: datetime.datetime | Unset = UNSET
    live_version: int | Unset = UNSET
    used_by_products: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_stats import TransactionalStats # noqa: PLC0415
        archived = self.archived

        attachments = self.attachments

        description = self.description

        draft_changes = self.draft_changes

        draft_version = self.draft_version

        id = self.id

        key = self.key

        name = self.name

        spark = self.spark



        stats = self.stats.to_dict()

        updated_at = self.updated_at.isoformat()

        category: dict[str, Any] | Unset = UNSET
        if not isinstance(self.category, Unset):
            category = self.category.to_dict()

        last_sent_at: str | Unset = UNSET
        if not isinstance(self.last_sent_at, Unset):
            last_sent_at = self.last_sent_at.isoformat()

        live_version = self.live_version

        used_by_products = self.used_by_products


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archived": archived,
            "attachments": attachments,
            "description": description,
            "draftChanges": draft_changes,
            "draftVersion": draft_version,
            "id": id,
            "key": key,
            "name": name,
            "spark": spark,
            "stats": stats,
            "updatedAt": updated_at,
        })
        if category is not UNSET:
            field_dict["category"] = category
        if last_sent_at is not UNSET:
            field_dict["lastSentAt"] = last_sent_at
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version
        if used_by_products is not UNSET:
            field_dict["usedByProducts"] = used_by_products

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_category_ref import TransactionalCategoryRef # noqa: PLC0415
        from ..models.transactional_stats import TransactionalStats # noqa: PLC0415
        d = dict(src_dict)
        archived = d.pop("archived")

        attachments = d.pop("attachments")

        description = d.pop("description")

        draft_changes = d.pop("draftChanges")

        draft_version = d.pop("draftVersion")

        id = d.pop("id")

        key = d.pop("key")

        name = d.pop("name")

        spark = cast(list[int], d.pop("spark"))


        stats = TransactionalStats.from_dict(d.pop("stats"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _category = d.pop("category", UNSET)
        category: TransactionalCategoryRef | Unset
        if isinstance(_category,  Unset):
            category = UNSET
        else:
            category = TransactionalCategoryRef.from_dict(_category)




        _last_sent_at = d.pop("lastSentAt", UNSET)
        last_sent_at: datetime.datetime | Unset
        if isinstance(_last_sent_at,  Unset):
            last_sent_at = UNSET
        else:
            last_sent_at = datetime.datetime.fromisoformat(_last_sent_at)




        live_version = d.pop("liveVersion", UNSET)

        used_by_products = d.pop("usedByProducts", UNSET)

        transactional_template_list_item = cls(
            archived=archived,
            attachments=attachments,
            description=description,
            draft_changes=draft_changes,
            draft_version=draft_version,
            id=id,
            key=key,
            name=name,
            spark=spark,
            stats=stats,
            updated_at=updated_at,
            category=category,
            last_sent_at=last_sent_at,
            live_version=live_version,
            used_by_products=used_by_products,
        )

        return transactional_template_list_item

