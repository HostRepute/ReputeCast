package segments

// The holdout group (M63 G-33): "In the holdout group" is yes or no, for
// the open period, whatever it holds contacts out of. The contacts view has
// it as in_holdout, so SQL segments read it too; it is computed only when a
// query reads it.

// FieldHoldout is the holdout group field.
const FieldHoldout = "in_holdout"

func init() {
	builtinFields[FieldHoldout] = fieldSpec{KindEnum, []string{"is"}}
	enumValues[FieldHoldout] = [][2]string{{"yes", ""}, {"no", ""}}
	builtinOrder = append(builtinOrder, FieldHoldout)
	extraCompile[FieldHoldout] = func(t *sqlText, c SegmentCondition) {
		if c.Value == "no" {
			t.raw("NOT c.in_holdout")
			return
		}
		t.raw("c.in_holdout")
	}
}

// holdoutColumns add in_holdout to the contacts view.
const holdoutColumns = `,
           EXISTS (SELECT 1 FROM public.holdout_periods hp
             WHERE hp.workspace_id = c.workspace_id AND hp.ended_at IS NULL
               AND public.rc_holdout_bucket(c.id, hp.seed) < hp.basis_points) AS in_holdout`
