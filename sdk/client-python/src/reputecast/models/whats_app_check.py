from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_check_key import WhatsAppCheckKey
from ..models.whats_app_check_status import WhatsAppCheckStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_cost import WhatsAppCost
  from ..models.whats_app_limit_split import WhatsAppLimitSplit





T = TypeVar("T", bound="WhatsAppCheck")



@_attrs_define
class WhatsAppCheck:
    """ 
        Attributes:
            count (int):
            helpers (list[str]):
            key (WhatsAppCheckKey):
            lists (list[str]):
            missing (list[str]):
            status (WhatsAppCheckStatus):
            without (int):
            account_status (str | Unset):
            author (str | Unset):
            category (str | Unset):
            checked_at (datetime.datetime | Unset):
            cost (WhatsAppCost | Unset):
            previous_quality (str | Unset):
            quality (str | Unset):
            quality_at (datetime.datetime | Unset):
            split (WhatsAppLimitSplit | Unset):
            template (str | Unset):
            template_status (str | Unset):
     """

    count: int
    helpers: list[str]
    key: WhatsAppCheckKey
    lists: list[str]
    missing: list[str]
    status: WhatsAppCheckStatus
    without: int
    account_status: str | Unset = UNSET
    author: str | Unset = UNSET
    category: str | Unset = UNSET
    checked_at: datetime.datetime | Unset = UNSET
    cost: WhatsAppCost | Unset = UNSET
    previous_quality: str | Unset = UNSET
    quality: str | Unset = UNSET
    quality_at: datetime.datetime | Unset = UNSET
    split: WhatsAppLimitSplit | Unset = UNSET
    template: str | Unset = UNSET
    template_status: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_limit_split import WhatsAppLimitSplit # noqa: PLC0415
        count = self.count

        helpers = self.helpers



        key = self.key.value

        lists = self.lists



        missing = self.missing



        status = self.status.value

        without = self.without

        account_status = self.account_status

        author = self.author

        category = self.category

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        cost: dict[str, Any] | Unset = UNSET
        if not isinstance(self.cost, Unset):
            cost = self.cost.to_dict()

        previous_quality = self.previous_quality

        quality = self.quality

        quality_at: str | Unset = UNSET
        if not isinstance(self.quality_at, Unset):
            quality_at = self.quality_at.isoformat()

        split: dict[str, Any] | Unset = UNSET
        if not isinstance(self.split, Unset):
            split = self.split.to_dict()

        template = self.template

        template_status = self.template_status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "helpers": helpers,
            "key": key,
            "lists": lists,
            "missing": missing,
            "status": status,
            "without": without,
        })
        if account_status is not UNSET:
            field_dict["accountStatus"] = account_status
        if author is not UNSET:
            field_dict["author"] = author
        if category is not UNSET:
            field_dict["category"] = category
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if cost is not UNSET:
            field_dict["cost"] = cost
        if previous_quality is not UNSET:
            field_dict["previousQuality"] = previous_quality
        if quality is not UNSET:
            field_dict["quality"] = quality
        if quality_at is not UNSET:
            field_dict["qualityAt"] = quality_at
        if split is not UNSET:
            field_dict["split"] = split
        if template is not UNSET:
            field_dict["template"] = template
        if template_status is not UNSET:
            field_dict["templateStatus"] = template_status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_limit_split import WhatsAppLimitSplit # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        helpers = cast(list[str], d.pop("helpers"))


        key = WhatsAppCheckKey(d.pop("key"))




        lists = cast(list[str], d.pop("lists"))


        missing = cast(list[str], d.pop("missing"))


        status = WhatsAppCheckStatus(d.pop("status"))




        without = d.pop("without")

        account_status = d.pop("accountStatus", UNSET)

        author = d.pop("author", UNSET)

        category = d.pop("category", UNSET)

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _cost = d.pop("cost", UNSET)
        cost: WhatsAppCost | Unset
        if isinstance(_cost,  Unset):
            cost = UNSET
        else:
            cost = WhatsAppCost.from_dict(_cost)




        previous_quality = d.pop("previousQuality", UNSET)

        quality = d.pop("quality", UNSET)

        _quality_at = d.pop("qualityAt", UNSET)
        quality_at: datetime.datetime | Unset
        if isinstance(_quality_at,  Unset):
            quality_at = UNSET
        else:
            quality_at = datetime.datetime.fromisoformat(_quality_at)




        _split = d.pop("split", UNSET)
        split: WhatsAppLimitSplit | Unset
        if isinstance(_split,  Unset):
            split = UNSET
        else:
            split = WhatsAppLimitSplit.from_dict(_split)




        template = d.pop("template", UNSET)

        template_status = d.pop("templateStatus", UNSET)

        whats_app_check = cls(
            count=count,
            helpers=helpers,
            key=key,
            lists=lists,
            missing=missing,
            status=status,
            without=without,
            account_status=account_status,
            author=author,
            category=category,
            checked_at=checked_at,
            cost=cost,
            previous_quality=previous_quality,
            quality=quality,
            quality_at=quality_at,
            split=split,
            template=template,
            template_status=template_status,
        )

        return whats_app_check

