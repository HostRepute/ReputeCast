from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.suppression_item_scope import SuppressionItemScope
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.suppression_source import SuppressionSource





T = TypeVar("T", bound="SuppressionItem")



@_attrs_define
class SuppressionItem:
    """ 
        Attributes:
            address (str):
            at (datetime.datetime):
            context (str):
            group (bool):
            id (str):
            original_code (str):
            reasons (list[str]):
            rows (int):
            scope (SuppressionItemScope):
            source (SuppressionSource):
            campaign_id (str | Unset):
            import_id (str | Unset):
            upload (bool | Unset):
            uploaded_by (str | Unset):
     """

    address: str
    at: datetime.datetime
    context: str
    group: bool
    id: str
    original_code: str
    reasons: list[str]
    rows: int
    scope: SuppressionItemScope
    source: SuppressionSource
    campaign_id: str | Unset = UNSET
    import_id: str | Unset = UNSET
    upload: bool | Unset = UNSET
    uploaded_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.suppression_source import SuppressionSource # noqa: PLC0415
        address = self.address

        at = self.at.isoformat()

        context = self.context

        group = self.group

        id = self.id

        original_code = self.original_code

        reasons = self.reasons



        rows = self.rows

        scope = self.scope.value

        source = self.source.to_dict()

        campaign_id = self.campaign_id

        import_id = self.import_id

        upload = self.upload

        uploaded_by = self.uploaded_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "at": at,
            "context": context,
            "group": group,
            "id": id,
            "originalCode": original_code,
            "reasons": reasons,
            "rows": rows,
            "scope": scope,
            "source": source,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if import_id is not UNSET:
            field_dict["importId"] = import_id
        if upload is not UNSET:
            field_dict["upload"] = upload
        if uploaded_by is not UNSET:
            field_dict["uploadedBy"] = uploaded_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.suppression_source import SuppressionSource # noqa: PLC0415
        d = dict(src_dict)
        address = d.pop("address")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        context = d.pop("context")

        group = d.pop("group")

        id = d.pop("id")

        original_code = d.pop("originalCode")

        reasons = cast(list[str], d.pop("reasons"))


        rows = d.pop("rows")

        scope = SuppressionItemScope(d.pop("scope"))




        source = SuppressionSource.from_dict(d.pop("source"))




        campaign_id = d.pop("campaignId", UNSET)

        import_id = d.pop("importId", UNSET)

        upload = d.pop("upload", UNSET)

        uploaded_by = d.pop("uploadedBy", UNSET)

        suppression_item = cls(
            address=address,
            at=at,
            context=context,
            group=group,
            id=id,
            original_code=original_code,
            reasons=reasons,
            rows=rows,
            scope=scope,
            source=source,
            campaign_id=campaign_id,
            import_id=import_id,
            upload=upload,
            uploaded_by=uploaded_by,
        )

        return suppression_item

