from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_preview_on_conflict import ImportPreviewOnConflict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.import_duplicate_example import ImportDuplicateExample
  from ..models.import_invalid_group import ImportInvalidGroup
  from ..models.import_risky_group import ImportRiskyGroup
  from ..models.import_verification import ImportVerification





T = TypeVar("T", bound="ImportPreview")



@_attrs_define
class ImportPreview:
    """ 
        Attributes:
            computed_at (datetime.datetime):
            duplicates (list[ImportDuplicateExample]):
            import_count (int):
            invalid (int):
            invalid_groups (list[ImportInvalidGroup]):
            merged (int): Already in ReputeCast
            not_imported (int): Statuses mapped to Skip
            on_conflict (ImportPreviewOnConflict):
            risky (int):
            risky_groups (list[ImportRiskyGroup]):
            rows (int):
            suppress_count (int):
            suppressed (int): Unsubscribed, bounced or complained in the source
            will_import (int): New contacts, subscribed
            verification (ImportVerification | Unset):
     """

    computed_at: datetime.datetime
    duplicates: list[ImportDuplicateExample]
    import_count: int
    invalid: int
    invalid_groups: list[ImportInvalidGroup]
    merged: int
    not_imported: int
    on_conflict: ImportPreviewOnConflict
    risky: int
    risky_groups: list[ImportRiskyGroup]
    rows: int
    suppress_count: int
    suppressed: int
    will_import: int
    verification: ImportVerification | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_duplicate_example import ImportDuplicateExample # noqa: PLC0415
        from ..models.import_invalid_group import ImportInvalidGroup # noqa: PLC0415
        from ..models.import_risky_group import ImportRiskyGroup # noqa: PLC0415
        from ..models.import_verification import ImportVerification # noqa: PLC0415
        computed_at = self.computed_at.isoformat()

        duplicates = []
        for duplicates_item_data in self.duplicates:
            duplicates_item = duplicates_item_data.to_dict()
            duplicates.append(duplicates_item)



        import_count = self.import_count

        invalid = self.invalid

        invalid_groups = []
        for invalid_groups_item_data in self.invalid_groups:
            invalid_groups_item = invalid_groups_item_data.to_dict()
            invalid_groups.append(invalid_groups_item)



        merged = self.merged

        not_imported = self.not_imported

        on_conflict = self.on_conflict.value

        risky = self.risky

        risky_groups = []
        for risky_groups_item_data in self.risky_groups:
            risky_groups_item = risky_groups_item_data.to_dict()
            risky_groups.append(risky_groups_item)



        rows = self.rows

        suppress_count = self.suppress_count

        suppressed = self.suppressed

        will_import = self.will_import

        verification: dict[str, Any] | Unset = UNSET
        if not isinstance(self.verification, Unset):
            verification = self.verification.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "computedAt": computed_at,
            "duplicates": duplicates,
            "importCount": import_count,
            "invalid": invalid,
            "invalidGroups": invalid_groups,
            "merged": merged,
            "notImported": not_imported,
            "onConflict": on_conflict,
            "risky": risky,
            "riskyGroups": risky_groups,
            "rows": rows,
            "suppressCount": suppress_count,
            "suppressed": suppressed,
            "willImport": will_import,
        })
        if verification is not UNSET:
            field_dict["verification"] = verification

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_duplicate_example import ImportDuplicateExample # noqa: PLC0415
        from ..models.import_invalid_group import ImportInvalidGroup # noqa: PLC0415
        from ..models.import_risky_group import ImportRiskyGroup # noqa: PLC0415
        from ..models.import_verification import ImportVerification # noqa: PLC0415
        d = dict(src_dict)
        computed_at = datetime.datetime.fromisoformat(d.pop("computedAt"))




        duplicates = []
        _duplicates = d.pop("duplicates")
        for duplicates_item_data in (_duplicates):
            duplicates_item = ImportDuplicateExample.from_dict(duplicates_item_data)



            duplicates.append(duplicates_item)


        import_count = d.pop("importCount")

        invalid = d.pop("invalid")

        invalid_groups = []
        _invalid_groups = d.pop("invalidGroups")
        for invalid_groups_item_data in (_invalid_groups):
            invalid_groups_item = ImportInvalidGroup.from_dict(invalid_groups_item_data)



            invalid_groups.append(invalid_groups_item)


        merged = d.pop("merged")

        not_imported = d.pop("notImported")

        on_conflict = ImportPreviewOnConflict(d.pop("onConflict"))




        risky = d.pop("risky")

        risky_groups = []
        _risky_groups = d.pop("riskyGroups")
        for risky_groups_item_data in (_risky_groups):
            risky_groups_item = ImportRiskyGroup.from_dict(risky_groups_item_data)



            risky_groups.append(risky_groups_item)


        rows = d.pop("rows")

        suppress_count = d.pop("suppressCount")

        suppressed = d.pop("suppressed")

        will_import = d.pop("willImport")

        _verification = d.pop("verification", UNSET)
        verification: ImportVerification | Unset
        if isinstance(_verification,  Unset):
            verification = UNSET
        else:
            verification = ImportVerification.from_dict(_verification)




        import_preview = cls(
            computed_at=computed_at,
            duplicates=duplicates,
            import_count=import_count,
            invalid=invalid,
            invalid_groups=invalid_groups,
            merged=merged,
            not_imported=not_imported,
            on_conflict=on_conflict,
            risky=risky,
            risky_groups=risky_groups,
            rows=rows,
            suppress_count=suppress_count,
            suppressed=suppressed,
            will_import=will_import,
            verification=verification,
        )

        return import_preview

